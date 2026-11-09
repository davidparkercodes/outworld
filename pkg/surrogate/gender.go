package surrogate

import (
	"strings"
	"unicode"
)

const (
	Female  = "f"
	Male    = "m"
	Neutral = ""
)

var realFemale = strings.Fields(`
Abby Abigail Ada Adele Adriana Agatha Aisha Alexa Alexandra Alice Alicia Alison Allison Alma Amanda Amber Amelia Amy Ana
Anastasia Andrea Angela Angelica Anita Ann Anna Annabelle Anne Annette Annie Antonia April Ariel Ashley Astrid Audrey Aurora
Ava Barbara Beatrice Becky Belinda Bella Bernadette Beth Bethany Betty Beverly Bianca Bonnie Brenda Bridget Brittany Brooke
Camila Camille Candace Cara Carla Carmen Carol Caroline Carolyn Cassandra Catherine Cathy Cecilia Celia Charlotte Chelsea
Cheryl Chloe Christina Christine Cindy Claire Clara Claudia Colleen Connie Cora Courtney Cynthia Daisy Dana Daniela Danielle
Daphne Dawn Debbie Deborah Debra Delia Delphine Denise Diana Diane Dolores Donna Dora Doris Dorothy Edith Elaine Eleanor
Elena Eliza Elizabeth Ella Ellen Eloise Elsa Elsie Emilia Emily Emma Erica Erin Esther Eva Eve Evelyn Faith Fatima Felicity
Fiona Florence Frances Francesca Freya Gabriela Gabrielle Gemma Georgia Gina Giselle Gloria Grace Greta Gwen Hannah Harriet
Hazel Heather Heidi Helen Helena Henrike Hilary Holly Hope Ida Imogen Ines Ingrid Irene Iris Isabel Isabella Isla Ivy Jackie
Jacqueline Jade Jane Janet Janice Jasmine Jean Jeanne Jenna Jennifer Jenny Jessica Jill Joan Joanna Jocelyn Josephine Joy
Joyce Judith Judy Julia Julie Juliet June Juniper Karen Kate Katherine Kathleen Kathryn Katie Kayla Kelly Kim Kimberly
Kristen Laura Lauren Layla Leah Leila Lena Leocadia Lily Linda Lisa Lois Lorraine Louise Lucia Lucy Luna Lydia Lynn Mabel
Madeline Madison Maggie Mallory Mara Margaret Margot Maria Mariana Marie Marilyn Marisol Marjorie Marlene Martha Mary
Matilda Maya May Megan Melanie Melissa Mia Michelle Mildred Mireille Miriam Molly Monica Nadia Nancy Naomi Natalia Natalie
Nicole Nina Nora Norma Odalys Olga Olive Olivia Paige Pamela Patricia Paula Pauline Pearl Penelope Penny Phoebe Phyllis
Polly Priya Priyanka Rachel Ramona Rebecca Regina Renee Rita Roberta Rosa Rosalind Rose Rosemary Ruby Ruth Sabrina Sally
Samantha Sandra Sara Sarah Saoirse Savannah Scarlett Selena Serena Sharon Sheila Shirley Sienna Sofia Sonia Sophia Sophie
Stella Stephanie Summer Susan Suzanne Sylvia Tamara Tanya Teresa Theresa Tiffany Tina Valerie Vanessa Vera Veronica Victoria
Violet Virginia Vivian Wanda Wendy Whitney Willow Ximena Yasmin Yolanda Yusra Yvonne Zara Zoe Zoey
`)

var realMale = strings.Fields(`
Aaron Abdul Abe Abel Abelard Abraham Adam Adrian Ahmed Alan Albert Alberto Alec Alejandro Alex Alexander Alfred Ali Allen
Alvin Amir Andre Andrew Andy Angelo Anselm Anthony Antonio Archie Arnold Art Arthur Arturo Austin Barry Bart Bartholomew
Ben Benedikt Benjamin Bernard Bill Billy Bob Bobby Brad Bradley Brandon Brendan Brian Bruce Bruno Bryan Byron Callum Calvin
Carl Carlos Casey Cedric Chad Charles Charlie Chase Chester Chris Christian Christopher Clarence Clark Clifford Clint Clyde
Cody Colin Connor Conrad Corey Corwin Craig Curtis Dale Damian Damon Dan Daniel Danny Darius Darren Dave David Dean Dennis
Derek Desmond Dexter Diego Dmitri Dominic Don Donald Douglas Drew Duncan Dustin Dwayne Dylan Earl Eddie Edgar Edmund Eduardo
Edward Edwin Eli Elias Elijah Elliot Emeka Emil Emmanuel Enrique Eric Ernest Ethan Eugene Evan Ezra Felix Fenwick Fernando
Finn Francis Francisco Frank Franklin Fred Frederick Gabriel Gary Gavin Gene George Gerald Gilbert Glenn Gordon Graham Grant
Greg Gregory Gustavo Harold Harry Harvey Hector Henry Herbert Hugh Hugo Ian Isaac Ivan Jack Jacob Jake James Jamal Jared
Jason Javier Jeff Jeffrey Jeremy Jerome Jerry Jesse Jim Jimmy Joe Joel John Johnny Jon Jonathan Jorge Jose Joseph Josh
Joshua Juan Julian Justin Karl Keith Kenneth Kevin Kwabena Kyle Lance Larry Lawrence Leo Leon Leonard Leroy Lewis Liam
Lionel Logan Lorcan Louis Lucas Luis Luke Malcolm Manuel Marc Marco Marcus Mario Mark Martin Marvin Mason Matthew Max
Maxwell Michael Miguel Mike Miles Mitchell Mohammed Morgan Nathan Nathaniel Neil Nelson Nicholas Nick Noah Norman Oliver
Omar Oscar Osric Owen Pablo Patrick Paul Pedro Peter Philip Pierre Rafael Ralph Ramon Randall Randy Raul Ray Raymond Reggie
Ricardo Richard Rick Robert Roberto Rodney Roger Roland Roman Ron Ronald Ross Roy Ruben Russell Ryan Sam Samuel Scott Sean
Sebastian Sergio Seth Shane Simon Stanley Stefan Stephen Sterling Steve Steven Stuart Ted Terrence Thaddeus Theo Theodore
Thomas Tim Timothy Tobiah Toby Todd Tom Tommy Tony Travis Trevor Troy Tyler Victor Vincent Wade Walter Warren Wayne Wesley
Will William Willie Xavier Zachary
`)

var genderOf = map[string]string{}

func init() {
	for _, list := range []struct {
		names []string
		g     string
	}{{femaleGiven, Female}, {maleGiven, Male}, {realFemale, Female}, {realMale, Male}} {
		for _, n := range list.names {
			genderOf[strings.ToLower(n)] = list.g
		}
	}
	for _, n := range neutralGiven {
		genderOf[strings.ToLower(n)] = Neutral
	}
}

func GenderOf(name string) string { return genderOf[strings.ToLower(strings.TrimSpace(name))] }

func NormalizeGender(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "f", "female", "woman", "w":
		return Female
	case "m", "male", "man":
		return Male
	}
	return Neutral
}

func givenList(gender string) []string {
	switch gender {
	case Female:
		return femaleGiven
	case Male:
		return maleGiven
	}
	return neutralGiven
}

func (g *Generator) GivenNameFor(real, gender string, attempt int) string {
	list := givenList(gender)
	r := g.stream("given:"+gender, real, attempt)
	if attempt < 3*len(list) {
		return r.pick(list)
	}
	return r.syllableName()
}

var nickTails = []string{"", "y", "ie", "o", "a", "i", "s", "ey"}

func Nickname(alias, real, sur string, attempt int) string {
	al, rl, sr := []rune(strings.ToLower(alias)), []rune(strings.ToLower(real)), []rune(sur)
	if len(sr) < 3 {
		return capitalize(string(sr) + nickTails[1+attempt%(len(nickTails)-1)])
	}
	common := 0
	for common < len(al) && common < len(rl) && al[common] == rl[common] {
		common++
	}
	var stem []rune
	var tail string
	switch {
	case common >= 2:
		n := min(max(common, 2), len(sr)-1)
		stem, tail = sr[:n], string(al[common:])
	case strings.HasSuffix(string(rl), string(al)) && len(al) < len(rl):
		stem = sr[len(sr)-min(len(al), len(sr)-1):]
	default:
		stem, tail = sr[:min(3, len(sr)-1)], "y"
	}
	n := len(stem)
	if attempt >= len(nickTails) {
		grow := (attempt / len(nickTails)) % len(sr)
		if common >= 2 || len(stem) < len(sr) {
			n = min(len(sr), len(stem)+grow)
			if n == len(sr) {
				n = max(2, len(stem)-1)
			}
		}
		if common >= 2 {
			stem = sr[:n]
		}
	}
	out := string(stem) + tail + nickTails[attempt%len(nickTails)]
	if attempt >= 4*len(nickTails) {
		out = string(sr[:2]) + tail + nickTails[attempt%len(nickTails)] + string(rune('a'+attempt%26))
	}
	return capitalize(out)
}

func capitalize(s string) string {
	r := []rune(strings.ToLower(s))
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
