package iden3mobile

import (
	"errors"

	"github.com/iden3/go-iden3-core/merkletree"
)

// TODO. HAVE A LOOK AT CODE GEN
type BytesArray struct {
	array [][]byte
}

type byteEntrier struct {
	entry *merkletree.Entry
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

func (ba *BytesArray) toEntriers() ([]merkletree.Entrier, error) {
	entriers := []merkletree.Entrier{}
	for i := 0; i < ba.Len(); i++ {
		entry, err := merkletree.NewEntryFromBytes(ba.Get(i))
		if err != nil {
			return entriers, err
		}
		entriers = append(entriers, &byteEntrier{
			entry: entry,
		})
	}
	return entriers, nil
}

func (e *byteEntrier) Entry() *merkletree.Entry {
	return e.entry
}

type TicketsMap struct {
	m map[string]*Ticket
}

type TicketsMapInterface interface {
	F(*Ticket) error
}

func (tm *TicketsMap) Get(key string) (*Ticket, error) {
	if t, ok := tm.m[key]; ok {
		return t, nil
	}
	return &Ticket{}, errors.New("No ticket found kor the given key.")
}

func (tm *TicketsMap) ForEach(handler TicketsMapInterface) error {
	for _, t := range tm.m {
		mutex.Lock()
		if err := handler.F(t); err != nil {
			return err
		}
		mutex.Unlock()
	}
	return nil
}
