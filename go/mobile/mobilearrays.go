package iden3mobile

import (
	"errors"
	"sync"

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

// TODO: impl
// func (ba *BytesArray) toClaimers() ([]claims.Claimer, error) {
// 	claimers := []claims.Claimer{}
// 	for i := 0; i < ba.Len(); i++ {
// 		entry, err := merkletree.NewEntryFromBytes(ba.Get(i))
// 		if err != nil {
// 			return entriers, err
// 		}
// 		entriers = append(entriers, &byteEntrier{
// 			entry: entry,
// 		})
// 	}
// 	return entriers, nil
// }

func (e *byteEntrier) Entry() *merkletree.Entry {
	return e.entry
}

type TicketsMap struct {
	sync.RWMutex
	m          map[string]*Ticket
	shouldStop bool
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

func (tm *TicketsMap) Cancel(key string) error {
	if _, ok := tm.m[key]; ok {
		tm.Lock()
		delete(tm.m, key)
		tm.Unlock()
		return nil
	}
	return errors.New("No ticket found kor the given key.")
}

func (tm *TicketsMap) ForEach(handler TicketsMapInterface) error {
	tm.RLock()
	defer tm.RUnlock()
	for _, t := range tm.m {
		if err := handler.F(t); err != nil {
			return err
		}
	}
	return nil
}
