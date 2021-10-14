package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const input = `package test

import (
	"log"
	"time"

	test_package "github.com/SamuelTheJackson/justtrack-fmt/test/test-package"
	test_package "github.com/SamuelTheJackson/justtrack-fmt/test/testpackage"
)

const (
	i = "j"
	a = "a"
	c = "iota"

	e = iota
	q
	w
	s
	l
	y
	d
)

var (
	D = "j"
	C = "j"
	A = "a"
	U = func() string {
		return "asdf"
	}
)

type Dog interface {
	Bite(times int) error
	Bark(times int) error
}

type Fog2 interface {
	Dog
	Fogging(times int) error
}

type Cat struct {
	CreatedAt    time.Time
	Id           uint
	thisIsUint   uint32
	producer     Dog
	ThisIsStruct TestStruct
	Test         Dog
	Alone        chan int
}

type JustAStruct struct{}

type TestStruct struct {
	ThisIsString string
	repo         test_package.TestStruct
	B            JustAStruct
	Fog2
	CreatedAt    time.Time
	c            []byte
	ThisIsStruct time.Time
	Dog
	ThisIsUint uint32
	A          JustAStruct
	Id         uint
	TestStruct struct {
		Test string
		// this is the Id
		// pls keep it to this line
		Id        *uint
		CreatedAt time.Time
	}
	b uint32
	ThisIsStruct3 *JustAStruct
	// commentForStruct2
	ThisIsStruct2 JustAStruct
	C             JustAStruct
	// this is a comment for
	// the logger
	logger log.Logger
	Qual   *string
	Stal   *string
	a      string
	D      []JustAStruct
}

func H() {}

func test(test string) {
	if test == "" {
		return
	}
	for _, i := range []string{"", "jasdf", "Jsfd"} {
		if i == "" {
			continue
		}
		if i == "ja" {
			i = ""
			continue
		}
	}

	if test == "hello" {
		test = ""
		return
	} else {
		test = "d"
		return
	}

	return
}
`

const expectedOutput = `package test

import (
	"log"
	"time"

	"github.com/SamuelTheJackson/justtrack-fmt/test/test-package"
	test_package "github.com/SamuelTheJackson/justtrack-fmt/test/testpackage"
)

const (
	i = "j"
	a = "a"
	c = "iota"

	e = iota
	q
	w
	s
	l
	y
	d
)

var (
	A = "a"
	C = "j"
	D = "j"
	U = func() string {
		return "asdf"
	}
)

type Dog interface {
	Bark(times int) error
	Bite(times int) error
}

type Fog2 interface {
	Dog
	Fogging(times int) error
}

type Cat struct {
	Id           uint
	Alone        chan int
	producer     Dog
	Test         Dog
	ThisIsStruct TestStruct
	thisIsUint   uint32
	CreatedAt    time.Time
}

type JustAStruct struct{}

type TestStruct struct {
	Dog
	Fog2
	// this is a comment for
	// the logger
	logger     log.Logger
	Id         uint
	A          JustAStruct
	a          string
	B          JustAStruct
	b          uint32
	c          []byte
	C          JustAStruct
	D          []JustAStruct
	Qual       *string
	repo       test_package.TestStruct
	Stal       *string
	TestStruct struct {
		// this is the Id
		// pls keep it to this line
		Id        *uint
		Test      string
		CreatedAt time.Time
	}
	ThisIsString string
	ThisIsStruct time.Time
	// commentForStruct2
	ThisIsStruct2 JustAStruct
	ThisIsStruct3 *JustAStruct
	ThisIsUint    uint32
	CreatedAt     time.Time
}

func H() {}

func test(test string) {
	if test == "" {
		return
	}
	for _, i := range []string{"", "jasdf", "Jsfd"} {
		if i == "" {
			continue
		}
		if i == "ja" {
			i = ""

			continue
		}
	}

	if test == "hello" {
		test = ""

		return
	} else {
		test = "d"

		return
	}

	return
}
`

func Test_ProcessFile(t *testing.T) {
	in := strings.NewReader(input)
	var out bytes.Buffer

	err := FormatFile(in, &out)
	assert.NoError(t, err)

	assert.Equal(t, expectedOutput, out.String())
}
