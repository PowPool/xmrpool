package blocktemplate

import "io"

type TxInGen struct {
	Height uint32
}

func (t TxInGen) Pack(writer io.Writer) error {
	err := PackVarInt(writer, uint64(t.Height))
	if err != nil {
		return err
	}
	return nil
}

func (t *TxInGen) UnPack(reader io.Reader) error {
	r, err := UnPackVarInt(reader)
	if err != nil {
		return err
	}
	t.Height = uint32(r)
	return nil
}

type TxOutGen struct {
	Amount uint64
	PubKey CryptoPubKey
}

func (t TxOutGen) Pack(writer io.Writer) error {
	err := PackVarInt(writer, t.Amount)
	if err != nil {
		return err
	}
	err = t.PubKey.Pack(writer)
	if err != nil {
		return err
	}
	return nil
}

func (t *TxOutGen) UnPack(reader io.Reader) error {
	r, err := UnPackVarInt(reader)
	if err != nil {
		return err
	}
	t.Amount = r
	err = t.PubKey.UnPack(reader)
	if err != nil {
		return err
	}
	return nil
}

type MinerTransaction struct {
	Version    uint64
	UnlockTime uint64
	Vin        []TxInGen
	Vout       []TxOutGen
	Extra      []byte
	RctSigType uint8
}

func (m MinerTransaction) Pack(writer io.Writer) error {
	err := PackVarInt(writer, m.Version)
	if err != nil {
		return err
	}
	err = PackVarInt(writer, m.UnlockTime)
	if err != nil {
		return err
	}
	err = PackVarInt(writer, uint64(len(m.Vin)))
	if err != nil {
		return err
	}
	for _, vin := range m.Vin {
		err = vin.Pack(writer)
		if err != nil {
			return err
		}
	}
	err = PackVarInt(writer, uint64(len(m.Vout)))
	if err != nil {
		return err
	}
	for _, vout := range m.Vout {
		err = vout.Pack(writer)
		if err != nil {
			return err
		}
	}
	err = PackVarInt(writer, uint64(len(m.Extra)))
	if err != nil {
		return err
	}
	_, err = writer.Write(m.Extra)
	if err != nil {
		return err
	}
	err = PackVarInt(writer, uint64(m.RctSigType))
	if err != nil {
		return err
	}
	return nil
}

func (m *MinerTransaction) UnPack(reader io.Reader) error {
	r, err := UnPackVarInt(reader)
	if err != nil {
		return err
	}
	m.Version = r
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	m.UnlockTime = r
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	m.Vin = make([]TxInGen, int(r))
	for i := 0; i < int(r); i++ {
		err = m.Vin[i].UnPack(reader)
		if err != nil {
			return err
		}
	}
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	m.Vout = make([]TxOutGen, int(r))
	for i := 0; i < int(r); i++ {
		err = m.Vout[i].UnPack(reader)
		if err != nil {
			return err
		}
	}
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	m.Extra = make([]byte, r)
	_, err = reader.Read(m.Extra[0:r])
	if err != nil {
		return err
	}
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	m.RctSigType = uint8(r)
	return nil
}
