package iden3mobile

// TODO. HAVE A LOOK AT CODE GEN
type BytesArray struct {
	array [][]byte
}

func NewBytesArray() *BytesArray {
	return &BytesArray{
		array: make([][]byte, 0),
	}
}

func (ba *BytesArray) Len() int {
	return len(ba.array)
}

func (ba *BytesArray) Get(i int) []byte {
	return ba.array[i]
}

func (ba *BytesArray) Append(bs []byte) {
	ba.array = append(ba.array, bs)
}
