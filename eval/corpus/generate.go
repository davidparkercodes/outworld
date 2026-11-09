package corpus

import (
	"fmt"
	"math/rand/v2"
)

const (
	DefaultSeed   uint64 = 20261107
	PerTemplate          = 14
	CorpusVersion        = 1
)

func Generate(seed uint64) (*Corpus, []DictEntry) {
	r := rand.New(rand.NewPCG(seed, seed^0x5eed))
	g := &gen{r: r, people: buildPeople(r)}
	c := &Corpus{Version: CorpusVersion, Seed: seed}
	n := 0
	for range PerTemplate {
		for _, t := range templates {
			n++
			b := newBuilder(fmt.Sprintf("gen-%03d", n), "generated", t.name)
			b.system(TaskPrompt)
			t.fn(g, b)
			c.Items = append(c.Items, b.build())
		}
	}
	c.Items = append(c.Items, handwritten(g)...)
	return c, dictionary(g.people)
}

func People(seed uint64) []*Person {
	r := rand.New(rand.NewPCG(seed, seed^0x5eed))
	return buildPeople(r)
}
