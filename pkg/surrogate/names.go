package surrogate

import "strings"

var femaleGiven = strings.Fields(`
Abigail Adeline Agnes Alana Alessia Alina Amara Anika Annika Ariana Astrid Benita Bianca Bronwyn Camila Carmela Cecily
Celeste Corinne Cosima Dalia Daria Delphine Dulcie Elodie Eloise Esme Farida Fenella Freya Gemma Giulia Greta Helga Ilse
Imogen Ines Josefina Kaia Larissa Leonie Liesel Linnea Mabel Marisol Maxine Meredith Mirela Nadia Noemi Odette Orla Paloma
Perpetua Petra Ramona Rhiannon Rosalind Sabine Saoirse Sigrid Svea Tamsin Theodora Valentina Vesna Wilhelmina Yara Zelda
Zofia Beatrix Dagny Fiora Gwendolyn Isolde Katarina Marguerite Niamh Octavia Seraphina Ottilie Marcella Philippa
Clementine Evangeline Benedetta Gisela Leocadia Ondine Wendeline
`)

var maleGiven = strings.Fields(`
Aaron Adrian Aiden Alberto Aldo Alfie Alvaro Ambrose Anders Ansel Arlo Aurelio Benedikt Bertram Bodhi Callum Cassius Cedric
Cillian Clement Damon Dashiell Desmond Dimitri Dorian Eamon Edmund Emeric Enzo Evander Ezra Fabian Felix Finnian Florian
Gareth Gideon Gunnar Halvard Hamish Henrik Idris Isidore Ivo Jareth Jonas Kasimir Keegan Kenji Klaus Lachlan Leander Lorcan
Lucian Ludovic Lysander Magnus Malcolm Matthias Montague Nikolai Nils Oisin Olivier Osric Otto Phineas Quentin Rasmus
Reinhold Rufus Severin Silas Soren Stellan Tancredi Thaddeus Tobias Ulrich Viggo Xavier Yusuf Anselm Caspian Emmerich
Horatio Jasper Linus Peregrine Rafferty Torvald Ignatius Leopold Bartholomew Casimir Ferdinand Maximilian Percival Tiberius
`)

var neutralGiven = strings.Fields(`
Jules Remy Kiran Hadley Hartley Rowan Emery Ellis Sasha Quinn Avery Arden Blair Sage Reese Linden Wren Tatum Sloane Shiloh
Teagan Eden Lennox Oakley Peyton Rory Skyler Briar Dakota Finley Harper Kendall Marin Noel Riley Sutton Ainsley Darcy
Emerson Hollis Jamie Merritt Robin Sawyer Shea Tierney Alexis Blake Corey Dallas Ember Jody Kerry Lindsey Micah
`)

var givenNames = append(append(append([]string(nil), femaleGiven...), maleGiven...), neutralGiven...)

var familyNames = strings.Fields(`
Abernathy Achterberg Albrecht Amundsen Ashcombe Aubertin Balcombe Ballantyne Bartholdi Beaumont Bergstrand Blackwood Blomqvist
Bramwell Brennan Calloway Carrington Castellano Chamberlin Delacroix Desrosiers Dunmore Eastwick Eckhardt Ellingsen Engstrom
Fairbanks Falkner Fenwick Fitzgerald Fontaine Galbraith Garrity Gerhardt Gillespie Grimaldi Halloran Hargreaves Hartmann
Haverford Heathcote Hollister Holmberg Ingersoll Iversen Jablonski Kavanagh Kellerman Kincaid Kowalczyk Lachance Lambrecht
Langford Lindqvist Lockhart Lundgren Macintyre Mallory Marchetti Merriweather   Nakamura Navarro Northcott
Novak Oakhurst Okonkwo Oyelaran Pemberton Pennington Petrakis Quigley Radcliffe Ravenscroft Rasmussen Redfield Rinaldi
Rosenthal Rutherford Sandoval Santangelo Schaefer Sinclair  Stavros Sorensen Strickland Sutherland Talbot Thackeray
Thorne Tremblay Underhill Vanterpool Vasquez Vandermeer Waddington Wainwright Wakefield Whitlock Winterbourne Wolcott
Yamamoto Yardley Zabinski Zellweger Ashworth Bellamy Blakemore Castellanos Darrow Elsworth Fairweather Gallagher Hawthorne
Ibarra Jessup Kingsley  Marlowe   Prescott Quintero Rockwell Stanhope Tennyson Valdivia
Whitcombe Ainsworth Barrowclough Caldwell Dalrymple Etheridge Fairchild Greenhalgh Holloway Inglewood Jarrett 
Loughran Mackintosh Nettleton Oxley Pryce Rowntree Sheridan Thistlewood Upchurch Villeneuve Wetherby Yeardley Zimmerman
`)

var streetNames = strings.Fields(`
Alderbrook Ashgrove Birchmont Bramblewood Cedarfield Chestnut Clearwater Cobblestone Coppervale Cypress Dovecote Elmhurst
Fernhill Foxglove Glenmore Hawthorne Hazelwood Heatherfield Hollyhock  Kestrel Lakeview Larchmont Linden Magnolia
Maplecrest Meadowlark Millbrook Mistletoe Northgate Oakmere Orchard Pinecrest Primrose Quarry Ravenwood Redbud Riverside
Rosewood Sagebrush Sandpiper Silverbirch Southfield Stonebridge Sycamore Tamarack Thornbury Tidewater Valleyview Westbrook
Whispering Willowmere Windmill Wisteria Woodland Yarrow
`)

var cityNames = strings.Fields(`
Ashborough Bellhaven Birchfield Brookvale Cedarburg Clearbrook Coldwater Crestwood Dunmere Eastholm Elmstead Fairhollow
Fernvale Glenhaven Greystone Harborview Hollowmere Kingsbridge Lakemont Larkfield Maplestead Marshfield Meadowbrook Millhaven
Northbury Oakvale Pinehurst Quarrytown Ravensbrook Ridgemont Riverbend Rosedale Silverlake Southmere Stonehaven Thornfield
Westhollow Whitbury Willowbrook Woodmere
`)

var orgStems = strings.Fields(`
Aldergate Bluefen Brightwater Caldera Copperline Driftwood Emberly Fernway Glasswing Halcyon Ironleaf Juniperline Kestrel
Lanternfly  Meridian Northwind Oakhollow Pinewick Quillstone Riverstone Saltmarsh Silverfin Stillwater Tidewell
Umberfield Verdant Wayfarer Westmere Wildmoor Windrow Yellowpine Zephyrine Amberly Bramblecote Cobaltine Dawnridge Evenfall
`)

var orgKinds = strings.Fields(`Partners Group Holdings Labs Systems Works Collective Associates Studio Ventures Industries Logistics Health Analytics`)

var syllOnset = strings.Fields(`b br c ch d dr f fl g gr h j k kl l m n p pr r s sh st t th tr v w z`)
var syllVowel = strings.Fields(`a e i o u ai ea io ou ae ia`)
var syllCoda = []string{"", "", "n", "r", "l", "s", "th", "nd", "rk", "m", "x", "v"}

var fictionalGiven = map[string]bool{}
var fictionalFamily = map[string]bool{}

func init() {
	for _, n := range givenNames {
		fictionalGiven[strings.ToLower(n)] = true
	}
	for _, n := range familyNames {
		fictionalFamily[strings.ToLower(n)] = true
	}
}

func IsBundledGivenName(s string) bool { return fictionalGiven[strings.ToLower(s)] }

func IsBundledFamilyName(s string) bool { return fictionalFamily[strings.ToLower(s)] }
