package payouts

import (
	"errors"
	"fmt"
	"github.com/PowPool/xmrpool/cnutil"
	"github.com/PowPool/xmrpool/pool"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PowPool/xmrpool/rpc"
	"github.com/PowPool/xmrpool/storage"
	. "github.com/PowPool/xmrpool/util"
)

//const minDepth = 16
//const byzantiumHardForkHeight = 4370000
//const istanbulHardForkHeight = 7080000

//var homesteadReward = math.MustParseBig256("5000000000000000000")
//var byzantiumReward = math.MustParseBig256("3000000000000000000")
//var istanbulReward = math.MustParseBig256("2000000000000000000")

// Donate 10% from pool fees to developers
const donationFee = 0.0
const donationAccount = ""

type BlockUnlocker struct {
	config   *pool.UnlockerConfig
	backend  *storage.RedisClient
	rpc      *rpc.RPCClient
	halt     bool
	lastFail error

	// halt recover counter
	counter uint64
}

func NewBlockUnlocker(cfg *pool.UnlockerConfig, backend *storage.RedisClient) *BlockUnlocker {
	if len(cfg.PoolFeeAddress) != 0 && !cnutil.ValidateAddress(cfg.PoolFeeAddress) {
		Error.Fatalln("Invalid poolFeeAddress", cfg.PoolFeeAddress)
	}
	//if cfg.Depth < minDepth*2 {
	//	Error.Fatalf("Block maturity depth can't be < %v, your depth is %v", minDepth*2, cfg.Depth)
	//}
	//if cfg.ImmatureDepth < minDepth {
	//	Error.Fatalf("Immature depth can't be < %v, your depth is %v", minDepth, cfg.ImmatureDepth)
	//}
	u := &BlockUnlocker{config: cfg, backend: backend}
	rawUrl := fmt.Sprintf("http://%s:%v/json_rpc", cfg.DaemonHost, cfg.DaemonPort)
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil
	}
	rpcClient := &rpc.RPCClient{Name: cfg.DaemonName, Url: url}
	timeout, _ := time.ParseDuration(cfg.Timeout)
	rpcClient.SetClient(&http.Client{
		Timeout: timeout,
	})

	u.rpc = rpcClient
	u.counter = 0

	return u
}

func (u *BlockUnlocker) Start() {
	Info.Println("Starting block unlocker")
	intv := MustParseDuration(u.config.Interval)
	timer := time.NewTimer(intv)
	Info.Printf("Set block unlock interval to %v", intv)

	// Immediately unlock after start
	u.unlockPendingBlocks()
	u.unlockAndCreditMiners()
	timer.Reset(intv)

	go func() {
		for {
			select {
			case <-timer.C:
				u.unlockPendingBlocks()
				u.unlockAndCreditMiners()
				timer.Reset(intv)
			}
		}
	}()
}

type UnlockResult struct {
	maturedBlocks  []*storage.BlockData
	orphanedBlocks []*storage.BlockData
	orphans        int
	uncles         int
	blocks         int
}

func (u *BlockUnlocker) unlockCandidates(candidates []*storage.BlockData) (*UnlockResult, error) {
	result := &UnlockResult{}

	// Data row is: "nonce:enonce1:enonce2:timestamp:diff:totalShares:coinBaseValue:blkTotalFee"
	for _, candidate := range candidates {
		blockHeaderReply, err := u.rpc.GetBlockHeaderByHeight(candidate.Height)
		if err != nil {
			Error.Printf("Error while retrieving block %v from node: %v", candidate.Height, err)
			return nil, err
		}

		// check block nonce
		blockNonceHex := fmt.Sprintf("%08x", blockHeaderReply.BlockHeader.Nonce)

		// endian reverse
		blockNonceHexTmp := ""
		for i := 0; i < 4; i++ {
			blockNonceHexTmp = blockNonceHex[i*2:i*2+2] + blockNonceHexTmp
		}
		blockNonceHex = blockNonceHexTmp

		if len(candidate.Nonce) > 0 && strings.EqualFold(candidate.Nonce, blockNonceHex) {
			// check block reward
			totalReward := big.NewInt(0).Add(candidate.CoinBaseValue, candidate.BlkTotalFee).Int64()
			if totalReward != blockHeaderReply.BlockHeader.Reward {
				errMsg := fmt.Sprintf("Validate block total reward fail: reward from candidate[%d,%d], reward from block[%d]",
					candidate.CoinBaseValue.Int64(), candidate.BlkTotalFee.Int64(), blockHeaderReply.BlockHeader.Reward)
				u.halt = true
				u.lastFail = errors.New(errMsg)
				Error.Printf("%s", errMsg)
				return nil, err
			}

			err = u.handleBlock(blockHeaderReply.BlockHeader, candidate)
			if err != nil {
				u.halt = true
				u.lastFail = err
				return nil, err
			}
			result.blocks++
			result.maturedBlocks = append(result.maturedBlocks, candidate)
			Info.Printf("Mature block %v with %v tx, hash: %v", candidate.Height, blockHeaderReply.BlockHeader.NumTxes, candidate.Hash[0:10])
		} else {
			result.orphans++
			candidate.Orphan = true
			result.orphanedBlocks = append(result.orphanedBlocks, candidate)
			Info.Printf("Orphaned block %v:%v", candidate.RoundHeight, candidate.Nonce)
		}
	}
	return result, nil
}

func (u *BlockUnlocker) handleBlock(blockHeader rpc.BlockHeader, candidate *storage.BlockData) error {
	reward := new(big.Int).Set(candidate.CoinBaseValue)
	// Add TX fees
	extraTxReward := new(big.Int).Set(candidate.BlkTotalFee)

	if u.config.KeepTxFees {
		candidate.ExtraReward = new(big.Int).Set(extraTxReward)
	} else {
		reward.Add(reward, extraTxReward)
	}

	candidate.Orphan = false
	candidate.Hash = blockHeader.Hash
	candidate.Reward = new(big.Int).Set(reward)
	return nil
}

func (u *BlockUnlocker) unlockPendingBlocks() {
	if u.halt {
		if u.counter%3 == 0 {
			u.halt = false
			u.lastFail = nil
			Info.Println("Unlocking suspending recovered temporarily")
			goto SKIP1
		} else {
			u.counter = u.counter + 1
		}
		Info.Println("Unlocking suspended due to last critical error:", u.lastFail)
		return
	}

SKIP1:

	blockCountReply, err := u.rpc.GetBlockCount()
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Unable to get current block height from node: %v", err)
		return
	}

	candidates, err := u.backend.GetCandidates(blockCountReply.Count - u.config.ImmatureDepth)
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Failed to get block candidates from backend: %v", err)
		return
	}

	if len(candidates) == 0 {
		Info.Println("No block candidates to unlock")
		return
	}

	result, err := u.unlockCandidates(candidates)
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Failed to unlock blocks: %v", err)
		return
	}
	Info.Printf("Immature %v blocks, %v uncles, %v orphans", result.blocks, result.uncles, result.orphans)

	err = u.backend.WritePendingOrphans(result.orphanedBlocks)
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Failed to insert orphaned blocks into backend: %v", err)
		return
	} else {
		Info.Printf("Inserted %v orphaned blocks to backend", result.orphans)
	}

	totalRevenue := new(big.Rat)
	totalMinersProfit := new(big.Rat)
	totalPoolProfit := new(big.Rat)

	for _, block := range result.maturedBlocks {
		revenue, minersProfit, poolProfit, roundRewards, err := u.calculateRewards(block)
		if err != nil {
			u.halt = true
			u.lastFail = err
			Error.Printf("Failed to calculate rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		err = u.backend.WriteImmatureBlock(block, roundRewards)
		if err != nil {
			u.halt = true
			u.lastFail = err
			Error.Printf("Failed to credit rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		totalRevenue.Add(totalRevenue, revenue)
		totalMinersProfit.Add(totalMinersProfit, minersProfit)
		totalPoolProfit.Add(totalPoolProfit, poolProfit)

		logEntry := fmt.Sprintf(
			"IMMATURE %v: revenue %v, miners profit %v, pool profit: %v",
			block.RoundKey(),
			FormatRatReward(revenue),
			FormatRatReward(minersProfit),
			FormatRatReward(poolProfit),
		)
		entries := []string{logEntry}
		for login, reward := range roundRewards {
			entries = append(entries, fmt.Sprintf("\tREWARD %v: %v: %v Satoshi", block.RoundKey(), login, reward))
		}
		Info.Println(strings.Join(entries, "\n"))
	}

	Info.Printf(
		"IMMATURE SESSION: revenue %v, miners profit %v, pool profit: %v",
		FormatRatReward(totalRevenue),
		FormatRatReward(totalMinersProfit),
		FormatRatReward(totalPoolProfit),
	)
}

func (u *BlockUnlocker) unlockAndCreditMiners() {
	if u.halt {
		if u.counter%3 == 0 {
			u.halt = false
			u.lastFail = nil
			Info.Println("Unlocking suspending recovered temporarily")
			goto SKIP2
		} else {
			u.counter = u.counter + 1
		}
		Info.Println("Unlocking suspended due to last critical error:", u.lastFail)
		return
	}

SKIP2:

	blockCountReply, err := u.rpc.GetBlockCount()
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Unable to get current block height from node: %v", err)
		return
	}

	immature, err := u.backend.GetImmatureBlocks(blockCountReply.Count - u.config.Depth)
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Failed to get block candidates from backend: %v", err)
		return
	}

	if len(immature) == 0 {
		Info.Println("No immature blocks to credit miners")
		return
	}

	result, err := u.unlockCandidates(immature)
	if err != nil {
		u.halt = true
		u.lastFail = err
		Error.Printf("Failed to unlock blocks: %v", err)
		return
	}
	Info.Printf("Unlocked %v blocks, %v uncles, %v orphans", result.blocks, result.uncles, result.orphans)

	for _, block := range result.orphanedBlocks {
		err = u.backend.WriteOrphan(block)
		if err != nil {
			u.halt = true
			u.lastFail = err
			Error.Printf("Failed to insert orphaned block into backend: %v", err)
			return
		}
	}
	Info.Printf("Inserted %v orphaned blocks to backend", result.orphans)

	totalRevenue := new(big.Rat)
	totalMinersProfit := new(big.Rat)
	totalPoolProfit := new(big.Rat)

	for _, block := range result.maturedBlocks {
		revenue, minersProfit, poolProfit, roundRewards, err := u.calculateRewards(block)
		if err != nil {
			u.halt = true
			u.lastFail = err
			Error.Printf("Failed to calculate rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		err = u.backend.WriteMaturedBlock(block, roundRewards)
		if err != nil {
			u.halt = true
			u.lastFail = err
			Error.Printf("Failed to credit rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		totalRevenue.Add(totalRevenue, revenue)
		totalMinersProfit.Add(totalMinersProfit, minersProfit)
		totalPoolProfit.Add(totalPoolProfit, poolProfit)

		logEntry := fmt.Sprintf(
			"MATURED %v: revenue %v, miners profit %v, pool profit: %v",
			block.RoundKey(),
			FormatRatReward(revenue),
			FormatRatReward(minersProfit),
			FormatRatReward(poolProfit),
		)
		entries := []string{logEntry}
		for login, reward := range roundRewards {
			entries = append(entries, fmt.Sprintf("\tREWARD %v: %v: %v Satoshi", block.RoundKey(), login, reward))
		}
		Info.Println(strings.Join(entries, "\n"))
	}

	Info.Printf(
		"MATURE SESSION: revenue %v, miners profit %v, pool profit: %v",
		FormatRatReward(totalRevenue),
		FormatRatReward(totalMinersProfit),
		FormatRatReward(totalPoolProfit),
	)
}

func (u *BlockUnlocker) calculateRewards(block *storage.BlockData) (*big.Rat, *big.Rat, *big.Rat, map[string]int64, error) {
	revenue := new(big.Rat).SetInt(block.Reward)
	minersProfit, poolProfit := chargeFee(revenue, u.config.PoolFee)

	shares, err := u.backend.GetRoundShares(block.RoundHeight, block.Nonce)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rewards := calculateRewardsForShares(shares, block.TotalShares, minersProfit)

	if block.ExtraReward != nil {
		extraReward := new(big.Rat).SetInt(block.ExtraReward)
		poolProfit.Add(poolProfit, extraReward)
		revenue.Add(revenue, extraReward)
	}

	if u.config.Donate {
		//var donation = new(big.Rat)
		//poolProfit, donation = chargeFee(poolProfit, donationFee)
		//login := strings.ToLower(donationAccount)
		//rewards[login] += weiToShannonInt64(donation)
	}

	if len(u.config.PoolFeeAddress) != 0 {
		address := strings.ToLower(u.config.PoolFeeAddress)
		value, _ := strconv.ParseInt(poolProfit.FloatString(0), 10, 64)
		rewards[address] += value
	}

	return revenue, minersProfit, poolProfit, rewards, nil
}

func calculateRewardsForShares(shares map[string]int64, total int64, reward *big.Rat) map[string]int64 {
	rewards := make(map[string]int64)

	for login, n := range shares {
		percent := big.NewRat(n, total)
		workerReward := new(big.Rat).Mul(reward, percent)
		value, _ := strconv.ParseInt(workerReward.FloatString(0), 10, 64)
		rewards[login] += value
	}
	return rewards
}

// Returns new value after fee deduction and fee value.
func chargeFee(value *big.Rat, fee float64) (*big.Rat, *big.Rat) {
	feePercent := new(big.Rat).SetFloat64(fee / 100)
	feeValue := new(big.Rat).Mul(value, feePercent)
	return new(big.Rat).Sub(value, feeValue), feeValue
}
