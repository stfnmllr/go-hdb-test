// SPDX-FileCopyrightText: 2020 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Prm represents a test parameter consisting of BatchCount and BatchSize
type Prm struct {
	BatchCount, BatchSize int
}

// PrmValue represents a flag Value fpr parameters.
type PrmValue struct {
	Prms []Prm
}

// String implements the flag.Value interface.
func (v *PrmValue) String() string {
	b := new(bytes.Buffer)
	last := len(v.Prms) - 1
	for i, prm := range v.Prms {
		b.WriteString(strconv.Itoa(prm.BatchCount))
		b.WriteString("x")
		b.WriteString(strconv.Itoa(prm.BatchSize))
		if i != last {
			b.WriteString(" ")
		}
	}
	return b.String()
}

// Set implements the flag.Value interface.
func (v *PrmValue) Set(s string) error {
	if v.Prms == nil {
		v.Prms = []Prm{}
	} else {
		v.Prms = v.Prms[:0]
	}

	for _, ts := range strings.Split(s, " ") {
		t := strings.Split(ts, "x")
		if len(t) != 2 {
			return fmt.Errorf("invalid value: %s", s)
		}
		var err error
		var prm Prm
		prm.BatchCount, err = strconv.Atoi(t[0])
		if err != nil {
			return err
		}
		prm.BatchSize, err = strconv.Atoi(t[1])
		if err != nil {
			return err
		}
		v.Prms = append(v.Prms, prm)
	}
	return nil
}

// ToNumRecordList returns a list of lists of prms with equal number of records.
func (v *PrmValue) ToNumRecordList() [][]Prm {

	// create categories by number of records
	m := make(map[int][]Prm)

	for _, prm := range v.Prms {
		numRecord := prm.BatchCount * prm.BatchSize
		m[numRecord] = append(m[numRecord], prm)
	}
	s := []int{}
	for numRecord := range m {
		s = append(s, numRecord)
	}
	// sort by number of records
	sort.Ints(s)

	r := make([][]Prm, len(s))
	for i, numRecord := range s {
		r[i] = m[numRecord]
	}
	return r
}
