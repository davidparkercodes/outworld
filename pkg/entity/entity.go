package entity

import (
	"fmt"
	"strings"
)

type Type string

const (
	Person    Type = "person"
	Email     Type = "email"
	Phone     Type = "phone"
	Address   Type = "address"
	Date      Type = "date"
	Org       Type = "org"
	Location  Type = "location"
	URL       Type = "url"
	IP        Type = "ip"
	Secret    Type = "secret"
	GovID     Type = "gov_id"
	Financial Type = "financial"
)

var AllTypes = []Type{Person, Email, Phone, Address, Date, Org, Location, URL, IP, Secret, GovID, Financial}

func ParseType(s string) (Type, error) {
	t := Type(strings.ToLower(strings.TrimSpace(s)))
	switch t {
	case "people", "per", "name":
		return Person, nil
	case "organization", "organisation", "company":
		return Org, nil
	case "place", "loc", "city":
		return Location, nil
	}
	for _, k := range AllTypes {
		if k == t {
			return t, nil
		}
	}
	return "", fmt.Errorf("unknown entity type %q", s)
}

type Action string

const (
	Keep       Action = "keep"
	Mask       Action = "mask"
	Shift      Action = "shift"
	Generalize Action = "generalize"
	Strip      Action = "strip"
)

func ParseAction(s string) (Action, error) {
	a := Action(strings.ToLower(strings.TrimSpace(s)))
	switch a {
	case Keep, Mask, Shift, Generalize, Strip:
		return a, nil
	case "surrogate":
		return Mask, nil
	case "redact", "drop":
		return Strip, nil
	}
	return "", fmt.Errorf("unknown policy action %q", s)
}

type Span struct {
	Start       int
	End         int
	Type        Type
	Label       string
	Text        string
	Source      string
	Priority    int
	Score       float64
	Canonical   string
	Generalized string
	Gender      string
}

func (s Span) Len() int { return s.End - s.Start }

func (s Span) Overlaps(o Span) bool { return s.Start < o.End && o.Start < s.End }
