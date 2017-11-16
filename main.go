package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

var (
	cmdIssueWarnings = flag.Bool("warn", false, "issue warnings in addition to errors")
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lpvet f.lp [f.lp...]")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetPrefix("lpvet: ")
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	issuedMesg := false
	for _, p := range flag.Args() {
		err, issued := vet(p, *cmdIssueWarnings)
		issuedMesg = issuedMesg || issued
		if err != nil {
			log.Print(err)
		}
	}

	if issuedMesg {
		os.Exit(1)
	}
}

type LP struct {
	Objective    Section
	Constraints  Section
	Bounds       Section
	GeneralVars  Section
	BinaryVars   Section
	SemiContVars Section
}

type Section struct {
	syms   []Symbol
	symSet map[string]bool
}

func (s *Section) AddSym(sym Symbol) {
	if s.symSet == nil {
		s.symSet = make(map[string]bool)
	}
	s.syms = append(s.syms, sym)
	s.symSet[sym.Value] = true
}

func (s *Section) Syms() []Symbol { return s.syms }
func (s *Section) HasSym(sym Symbol) bool {
	if s.symSet == nil {
		return false
	}
	return s.symSet[sym.Value]
}

type Symbol struct {
	Value string
	Pos   Pos
}

type Pos struct {
	File string
	Line int32
}

func (p Pos) String() string {
	return p.File + ":" + strconv.Itoa(int(p.Line))
}

const (
	MaxLineLen           = 510
	MaxVarLen            = 255
	MaxConstraintNameLen = MaxVarLen
)

func validVarName(n string) bool {
	// implement
	for _, c := range n {
		switch {
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case '0' <= c && c <= '9':
		default:
			switch c {
			case '!', '"', '#', '$', '%', '&', '(', ')', ',', '.', ';', '?', '@', '_', '‘', '\'', '{', '}', '~':
			default:
				return false
			}
		}
	}
	return true
}

func loadLP(p string) (*LP, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		lp     LP
		curSec *Section
	)
	pos := Pos{File: p}
	s := bufio.NewScanner(f)
	for s.Scan() {
		pos.Line++
		if len(s.Text()) > MaxLineLen {
			return nil, fmt.Errorf("%s: line too long (%d > %d)", pos, len(s.Text()), MaxLineLen)
		}
		t := strings.TrimSpace(s.Text())
		fields := strings.Fields(t)
		var h string
		if len(fields) >= 1 {
			h = strings.ToUpper(fields[0])
		}
		switch {
		case t == "":
			continue
		case strings.HasPrefix(t, "\\"):
			continue
		default:
			switch h {
			case "MIN", "MAX", "MINIMIZE", "MAXIMIZE", "MINIMUM", "MAXIMUM":
				curSec = &lp.Objective
				continue
			case "SUBJECT", "S.T", "SUCH", "ST", "ST.":
				curSec = &lp.Constraints
				continue
			case "BOUNDS", "BOUND":
				curSec = &lp.Bounds
				continue
			case "GENERAL", "GEN", "GENERALS":
				curSec = &lp.GeneralVars
				continue
			case "BINARY", "BIN", "BINARIES":
				curSec = &lp.BinaryVars
				continue
			case "SEMI-CONTINUOUS", "SEMI", "SEMIS":
				curSec = &lp.SemiContVars
				continue
			case "END":
				curSec = nil
				continue
			}
		}
		ci := strings.IndexByte(t, ':')
		if ci < 0 {
			ci = 0
		}
		t = t[ci:]
		fields = strings.FieldsFunc(t, func(r rune) bool {
			if unicode.IsSpace(r) {
				return true
			}
			switch r {
			case '+', '-', '=', '>', '<':
				return true
			}
			return false
		})
		if curSec == nil {
			return nil, fmt.Errorf("%s: not in a section", pos)
		}
		// Remaining fields are either symbols or numerals.
		// Assume if starts with letter or _, symbol.
		for _, f := range fields {
			// Not unicode safe. CPLEX isn't either.
			if unicode.IsLetter(rune(f[0])) || f[0] == '_' {
				if f == "inf" && curSec == &lp.Bounds {
					continue
				}
				if len(f) > MaxVarLen {
					return nil, fmt.Errorf("%s: variable too long: %q (%d > %d)", pos, f, len(f), MaxVarLen)
				}
				if !validVarName(f) {
					return nil, fmt.Errorf("%s: invalid variable name: %q", pos, f)
				}
				curSec.AddSym(Symbol{
					Value: f,
					Pos:   pos,
				})
			}
		}
	}
	return &lp, s.Err()
}

func vet(p string, issueWarnings bool) (error, bool) {
	lp, err := loadLP(p)
	if err != nil {
		return err, false
	}

	issued := false

	issuedFor := make(map[string]bool)

	issue := func(format string, s Symbol) {
		if !issuedFor[s.Value] {
			log.Printf(format, s.Pos, s.Value)
			issued = true
			issuedFor[s.Value] = true
		}
	}

	for _, sym := range lp.Objective.Syms() {
		if !lp.GeneralVars.HasSym(sym) && !lp.BinaryVars.HasSym(sym) && !lp.SemiContVars.HasSym(sym) {
			issue("%s: error: no var declaration for %s", sym)
		}
	}

	for _, sym := range lp.Constraints.Syms() {
		if !lp.GeneralVars.HasSym(sym) && !lp.BinaryVars.HasSym(sym) && !lp.SemiContVars.HasSym(sym) {
			issue("%s: error: no var declaration for %s", sym)
		}
	}

	for _, sym := range lp.Bounds.Syms() {
		if !lp.GeneralVars.HasSym(sym) && !lp.BinaryVars.HasSym(sym) && !lp.SemiContVars.HasSym(sym) {
			issue("%s: error: no var declaration for %s", sym)
		}
	}

	if issueWarnings {
		for _, sym := range lp.GeneralVars.Syms() {
			if !lp.Objective.HasSym(sym) && !lp.Constraints.HasSym(sym) {
				issue("%s: warning: no use of general var %s", sym)
			}
		}

		for _, sym := range lp.BinaryVars.Syms() {
			if !lp.Objective.HasSym(sym) && !lp.Constraints.HasSym(sym) {
				issue("%s: warning: no use of binary var %s", sym)
			}
		}

		for _, sym := range lp.SemiContVars.Syms() {
			if !lp.Objective.HasSym(sym) && !lp.Constraints.HasSym(sym) {
				issue("%s: warning: no use of semi-continuous var %s", sym)
			}
		}
	}
	return nil, issued
}
