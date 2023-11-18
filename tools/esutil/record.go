package main

import (
	"bytes"
	"encoding/json"
	"io"
)

type Record interface {
	DocID() string
	Body() io.ReadSeeker
}

func NewUpdateRecord(v Record) *UpdateRecord {
	a, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return &UpdateRecord{
		Document:    a,
		DocAsUpsert: true,
		inner:       v,
	}
}

type UpdateRecord struct {
	Document    json.RawMessage `json:"doc"`
	DocAsUpsert bool            `json:"doc_as_upsert"`

	inner Record `json:"-"`
}

func (u *UpdateRecord) DocID() string {
	return u.inner.DocID()
}

func (u *UpdateRecord) Body() io.ReadSeeker {
	body, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(body)
}
