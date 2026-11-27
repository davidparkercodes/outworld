package textutil

import "strings"

var HyphenVariants = []rune{0x2010, 0x2011, 0x2212, 0xFE63, 0xFF0D}

var RangeOrNameDashes = []rune{0x2012, 0x2013}

func IsHyphen(r rune) bool { return r == '-' || IsHyphenVariant(r) }

func IsRangeOrNameDash(r rune) bool { return r == 0x2012 || r == 0x2013 }

func IsNameJoiner(r rune) bool { return IsHyphen(r) || IsRangeOrNameDash(r) }

func HyphenFields(s string) []string { return strings.FieldsFunc(s, IsHyphen) }

func NameJoinerFields(s string) []string { return strings.FieldsFunc(s, IsNameJoiner) }
