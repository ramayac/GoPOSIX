package common

// compiledSpec is a pre-compiled lookup table for a FlagSpec.
// Built once per spec, cached on FlagSpec.compiled.
type compiledSpec struct {
	shortIdx    [128]int8 // byte → index into defs, or -1
	longIdx     []longEntry
	defs        []FlagDef
	stopAtFirst bool
}

type longEntry struct {
	name  string
	index uint8
}

func (s *FlagSpec) getOrCompile() *compiledSpec {
	if s.compiled != nil {
		return s.compiled
	}
	cs := s.compile()
	s.compiled = cs
	return cs
}

func (s *FlagSpec) compile() *compiledSpec {
	cs := &compiledSpec{
		defs:        s.Defs,
		stopAtFirst: s.StopAtFirstNonFlag,
	}
	for i := range cs.shortIdx {
		cs.shortIdx[i] = -1
	}
	if len(s.Defs) == 0 {
		return cs
	}
	longEntries := make([]longEntry, 0, len(s.Defs))
	for i, d := range s.Defs {
		if d.Short != "" && d.Short[0] < 128 {
			cs.shortIdx[d.Short[0]] = int8(i)
		}
		if d.Long != "" {
			longEntries = append(longEntries, longEntry{name: d.Long, index: uint8(i)})
		}
	}
	cs.longIdx = longEntries
	return cs
}

func (cs *compiledSpec) lookupShort(b byte) *FlagDef {
	if b >= 128 {
		return nil
	}
	idx := cs.shortIdx[b]
	if idx < 0 {
		return nil
	}
	return &cs.defs[idx]
}

func (cs *compiledSpec) lookupLong(name string) *FlagDef {
	for i := range cs.longIdx {
		if cs.longIdx[i].name == name {
			return &cs.defs[cs.longIdx[i].index]
		}
	}
	return nil
}
