package surrogate

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

type addrPlace struct {
	name, code string
}

type addrLocale struct {
	id        string
	typeFirst bool
	types     map[string]bool
	streets   map[string][]string
	stems     []string
	places    []addrPlace
	alt       []addrPlace
	given     []string
	family    []string
	titles    []string
	titleSlot map[string]bool
	cityLead  []string
	cityTail  []string
}

func parsePlaces(s string) []addrPlace {
	var out []addrPlace
	for _, f := range strings.Fields(s) {
		name, code, _ := strings.Cut(f, ":")
		out = append(out, addrPlace{name, code})
	}
	return out
}

func slotLists(pairs ...string) map[string][]string {
	m := map[string][]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		words := strings.Fields(pairs[i+1])
		if pairs[i] == "" {
			m[""] = words
		}
		for _, k := range strings.Fields(pairs[i]) {
			m[k] = words
		}
	}
	return m
}

var addrFR = &addrLocale{
	id: "fr", typeFirst: true,
	given:     strings.Fields(`Victor Jean Pierre Émile Louis Henri Paul Jules Léon Charles Georges Marcel André Albert Anatole Aristide Auguste Edmond Gustave Félix Étienne Eugène`),
	titles:    strings.Fields(`Général Maréchal Docteur Président Colonel Capitaine Commandant Professeur Cardinal Sergent Lieutenant Chanoine`),
	titleSlot: addrWordSet(`du le`),
	cityLead:  strings.Fields(`Saint Mont Port Bourg Château Villeneuve Pont Val Bois Neuville`),
	cityTail:  strings.Fields(`Loire Seine Marne Saône Garonne Dordogne Charente Vienne Moselle Meuse Oise Aube Isère Orne Eure Ain Aisne Allier`),
	types:     addrWordSet(`rue boulevard bd chemin allée allee impasse quai avenue av place cours route passage square`),
	streets: slotLists(
		"", `Pasteur Voltaire Molière Racine Lamartine Gambetta Carnot Diderot Montaigne Colbert Balzac Corneille Descartes Rabelais Michelet Berlioz Ampère Lavoisier Kléber Hoche Raspail Monge Laplace Buffon Béranger Ronsard Marceau Mirabeau Fénelon Lakanal`,
		"de", `Verdun Lorraine Bretagne Provence Normandie Bourgogne Champagne Picardie Touraine Savoie Flandre Gascogne Navarre Valmy Marengo Wagram Solférino Bellevue Beauséjour Belfort Colmar Lorette Montreuil Châteaudun`,
		"des les", `Lilas Acacias Tilleuls Roses Écoles Vignes Peupliers Érables Mésanges Alouettes Cerisiers Marronniers Jardins Moulins Prés Champs Fauvettes Hirondelles Bleuets Primevères Sablons Carrières Capucines Glycines Noisetiers Chênes Platanes Violettes Pinsons Saules`,
		"du le", `Moulin Château Stade Pont Marché Lavoir Parc Port Verger Collège Puits Four Calvaire Presbytère Coteau Couvent Canal Commerce Faubourg Temple Bois Midi Levant Ruisseau Clos`,
		"la", `Gare République Fontaine Mairie Liberté Paix Poste Forêt Chapelle Source Colline Prairie Plaine Vallée Croix Halle Tour Victoire Fraternité Résistance Libération Garenne Cerisaie Roseraie Tuilerie`,
		"l'", `Église Europe Abbaye Étang Écluse Orangerie Avenir Industrie Espérance Union Égalité Ormeau Arsenal Observatoire Aqueduc Ermitage Oratoire Hôpital Hermitage Usine`,
		"d'", `Alsace Anjou Artois Auvergne Aquitaine Orléans Austerlitz Italie Espagne Iéna Ulm Arcole Angleterre Islande Irlande Athènes`,
	),
	places: parsePlaces(`Angers:49000 Tours:37000 Reims:51100 Dijon:21000 Besançon:25000 Orléans:45000 Poitiers:86000 Limoges:87000 Amiens:80000 Rouen:76000 Caen:14000 Nancy:54000 Metz:57000 Mulhouse:68100 Colmar:68000 Annecy:74000 Chambéry:73000 Valence:26000 Avignon:84000 Nîmes:30000 Perpignan:66000 Béziers:34500 Pau:64000 Bayonne:64100 Angoulême:16000 Niort:79000 Vannes:56000 Quimper:29000 Lorient:56100 Brest:29200 Laval:53000 Chartres:28000 Troyes:10000 Auxerre:89000 Nevers:58000 Bourges:18000 Périgueux:24000 Agen:47000 Albi:81000 Tarbes:65000 Montauban:82000 Cahors:46000 Rodez:12000 Aurillac:15000 Épinal:88000 Belfort:90000 Mâcon:71000 Roanne:42300 Arras:62000 Beauvais:60000`),
}

var addrES = &addrLocale{
	id: "es", typeFirst: true,
	given:     strings.Fields(`Juan Pedro Luis Antonio Manuel Francisco Miguel Rafael Ramón Fernando Joaquín Andrés Vicente Ignacio Tomás Jaime Alfonso Rodrigo Gonzalo Agustín Emilio Julián`),
	titles:    strings.Fields(`General Doctor Maestro Padre Capitán Coronel Obispo Marqués Conde Duque Cardenal Almirante Profesor Pintor Poeta`),
	titleSlot: addrWordSet(`del el`),
	cityLead:  strings.Fields(`San Santa Villa Puerto Ciudad Castro Torre Monte Puente Villanueva`),
	cityTail:  strings.Fields(`Henares Duero Ebro Tajo Segura Tormes Jalón Guadiana Campos Arriba Abajo Júcar Sil Miño Esla Arlanza`),
	types:     addrWordSet(`calle carrer avenida avda av avinguda paseo passeig ronda camino carretera rambla plaza plaça glorieta travesía travesia`),
	streets: slotLists(
		"de", `Cervantes Goya Velázquez Quevedo Góngora Zurbarán Murillo Colón Lepanto Bailén Covadonga Numancia Sagunto Pelayo Jovellanos Galdós Unamuno Machado Albéniz Granados Sorolla Calderón Alarcón Espronceda Bécquer Zorrilla Balmes Larra Echegaray Moratín`,
		"del el", `Olmo Carmen Pino Río Prado Molino Castillo Rosario Puente Mercado Pozo Roble Laurel Almendro Naranjo Rey Sol Mar Monte Huerto Álamo Fresno Cerezo Parque`,
		"la", `Paz Luna Rosa Fuente Iglesia Constitución Libertad Estación Concordia Esperanza Encina Vega Ribera Cruz Reina Victoria Alameda Aurora Amistad Merced Soledad Palma Sierra`,
		"los", `Olivos Almendros Naranjos Pinos Álamos Reyes Molinos Robles Cerezos Rosales Jardines Tilos Castaños Geranios`,
		"las", `Flores Rosas Huertas Fuentes Eras Acacias Moreras Cruces Palmeras Viñas Delicias Escuelas Camelias Azucenas`,
		"d'", `Aragó Urgell Aribau Entença Olot Igualada Osona Andorra Amposta Elx`,
		"l'", `Església Hospital Argenteria Estació Escorxador Om Alzina Olivera`,
	),
	places: parsePlaces(`Valladolid:47 Salamanca:37 Burgos:09 León:24 Zamora:49 Palencia:34 Segovia:40 Ávila:05 Soria:42 Logroño:26 Pamplona:31 Huesca:22 Teruel:44 Lleida:25 Girona:17 Tarragona:43 Albacete:02 Cuenca:16 Guadalajara:19 Toledo:45 Cáceres:10 Badajoz:06 Huelva:21 Cádiz:11 Córdoba:14 Jaén:23 Almería:04 Granada:18 Murcia:30 Oviedo:33 Santander:39 Lugo:27 Ourense:32 Pontevedra:36 Vitoria:01`),
}

var addrPT = &addrLocale{
	id: "pt", typeFirst: true,
	given:     strings.Fields(`João António Manuel José Francisco Luís Pedro Fernando Afonso Henrique Duarte Miguel Vasco Nuno Álvaro Diogo Gil Bartolomeu Rui Egas`),
	titles:    strings.Fields(`Doutor General Padre Capitão Marquês Conde Duque Infante Professor Comendador Almirante Bispo Visconde Brigadeiro`),
	titleSlot: addrWordSet(`do o`),
	cityLead:  strings.Fields(`São Santa Vila Castelo Monte Ponte Torre Póvoa Aldeia`),
	cityTail:  strings.Fields(`Douro Minho Tejo Mondego Sado Lima Cávado Serra Vouga Guadiana Zêzere Tâmega Sor Coura`),
	types:     addrWordSet(`rua travessa praça praca avenida av largo alameda estrada calçada`),
	streets: slotLists(
		"", `Camões Garrett Herculano Bocage Castilho Junqueiro Quental Magalhães Cabral Coutinho Albuquerque Saldanha Pombal Fontes Bordalo Malhoa Columbano Nobre Régio Torga`,
		"de", `Ceuta Diu Goa Timor Macau Moçambique Angola Olivença Aljubarrota Alcobaça Mafra Sagres Cedofeita Belém Alcântara Camões Santarém`,
		"do o", `Carmo Comércio Ouro Norte Sol Salitre Século Alecrim Loureiro Souto Pinheiro Monte Castelo Paço Mercado Moinho Poço Jardim Olival Rosário Forno Outeiro`,
		"da a", `Liberdade República Alegria Boavista Fonte Igreja Escola Paz Prata Madalena Estrela Graça Saudade Esperança Palma Ponte Quinta Fábrica Torre Vinha Lapa Glória`,
		"dos os", `Combatentes Bombeiros Pescadores Navegantes Fanqueiros Douradores Sapateiros Correeiros Lusíadas Loureiros Pinheiros Moinhos Carvalhos`,
		"das as", `Flores Rosas Oliveiras Laranjeiras Amoreiras Fontainhas Acácias Camélias Hortas Escolas Olarias Musas Gaivotas`,
	),
	places: parsePlaces(`Braga:4700 Guimarães:4800 Viseu:3500 Coimbra:3000 Aveiro:3800 Leiria:2400 Santarém:2000 Évora:7000 Beja:7800 Faro:8000 Setúbal:2900 Bragança:5300 Guarda:6300 Covilhã:6200 Tomar:2300 Lagos:8600 Tavira:8800 Portimão:8500 Chaves:5400 Barcelos:4750 Amarante:4600 Lamego:5100 Ovar:3880 Peniche:2520 Elvas:7350 Estremoz:7100 Portalegre:7300 Abrantes:2200 Espinho:4500 Fafe:4820`),
}

var addrIT = &addrLocale{
	id: "it", typeFirst: true,
	given:     strings.Fields(`Giuseppe Giovanni Antonio Luigi Carlo Francesco Pietro Paolo Angelo Vincenzo Domenico Camillo Goffredo Ugo Enrico Alessandro Guglielmo Niccolò Silvio Massimo Cesare`),
	titles:    strings.Fields(`Cardinale Generale Dottor Professor Maresciallo Capitano Beato Conte Duca Principe`),
	titleSlot: addrWordSet(`del il`),
	cityLead:  strings.Fields(`San Santa Castel Monte Torre Villa Borgo Rocca Ponte`),
	cityTail:  strings.Fields(`Garda Piave Brenta Adda Ticino Arno Tevere Friuli Monferrato Chianti Adige Oglio Isonzo Emilia Umbria`),
	types:     addrWordSet(`via viale piazza piazzale corso vicolo largo lungomare`),
	streets: slotLists(
		"", `Verdi Garibaldi Mazzini Cavour Dante Manzoni Leopardi Carducci Pascoli Marconi Volta Galilei Colombo Rossini Puccini Bellini Donizetti Vivaldi Petrarca Foscolo Tasso Ariosto Alfieri Goldoni Giotto Tiziano Raffaello Boccaccio Parini Monti`,
		"di", `Francia Spagna Brera Novoli Ripoli Careggi Soffiano Peretola Quarto Terzolle Mezzo Sotto`,
		"del il", `Corso Lavoro Popolo Mare Porto Teatro Castello Mulino Giglio Sole Colle Campo Ponte Pozzo Duomo Carmine Risorgimento`,
		"della la", `Repubblica Libertà Vittoria Pace Stazione Chiesa Fontana Resistenza Costituzione Torre Rocca Valle Pineta Vigna Croce`,
		"delle le", `Rose Mimose Viole Ginestre Magnolie Querce Acacie Orchidee Camelie Fornaci Grazie Rimembranze`,
		"dei i", `Mille Tigli Pini Platani Gelsomini Ciliegi Pioppi Castagni Glicini Martiri Fiori Mulini Giardini`,
		"degli gli", `Ulivi Olmi Oleandri Aceri Abeti Artigiani Alpini Orti`,
		"dello lo", `Stadio Sport Statuto Scalo Speziale`,
		"l' d'", `Orso Arco Olmo Unione Indipendenza Industria Artigianato Aeroporto Università Acquedotto Annunziata Edera Usignolo`,
	),
	places: parsePlaces(`Verona:37121 Padova:35121 Vicenza:36100 Treviso:31100 Udine:33100 Trento:38122 Bolzano:39100 Brescia:25121 Bergamo:24121 Como:22100 Varese:21100 Pavia:27100 Cremona:26100 Mantova:46100 Parma:43121 Modena:41121 Ferrara:44121 Ravenna:48121 Rimini:47921 Pesaro:61121 Ancona:60121 Perugia:06121 Arezzo:52100 Siena:53100 Pisa:56121 Lucca:55100 Livorno:57121 Pistoia:51100 Novara:28100 Asti:14100 Cuneo:12100 Alessandria:15121 Savona:17100 Lecce:73100 Salerno:84121 Pescara:65121 Sassari:07100 Cagliari:09121`),
}

var addrDE = &addrLocale{
	id:       "de",
	given:    strings.Fields(`Friedrich Wilhelm Heinrich Karl Ludwig Johann Georg Konrad Otto Albert Ernst Theodor Gustav Max Robert Clara Käthe Bertha Carl Hermann Richard Franz`),
	family:   strings.Fields(`Schiller Goethe Lessing Mozart Beethoven Kant Humboldt Schubert Haydn Brahms Dürer Kepler Gutenberg Adenauer Ebert Heuss Bebel Fontane Storm Raabe Liebig Röntgen Siemens Daimler`),
	cityLead: strings.Fields(`Bad Sankt Neu Alt Groß Klein Ober Hohen Nieder Kirch`),
	cityTail: strings.Fields(`Main Rhein Neckar Donau Elbe Weser Saale Inn Lahn Ruhr Havel Oder Mosel Isar Lech Aller`),
	streets: slotLists(
		"", `Berliner Hamburger Münchner Kölner Frankfurter Leipziger Dresdner Bremer Stuttgarter Nürnberger Mainzer Kasseler Lübecker Rostocker Potsdamer Bonner Aachener Erfurter Weimarer Wiener Salzburger Zürcher Basler Berner`,
		"am im zum dem auf", `Mühlenbach Sonnenhang Kirchberg Rosengarten Stadtpark Waldrand Wiesengrund Schlossberg Weinberg Steinbruch Eichenhain Fischteich Sportplatz Mühlenteich Anger Hang Brink Kirchfeld Lindenhof Rathaus`,
		"der zur", `Mühle Kirche Linde Aue Wiese Heide Schmiede Ziegelei Bleiche Burg Schanze Kapelle Quelle Brücke Stadtmauer Lehmkuhle`,
		"den", `Linden Eichen Birken Wiesen Gärten Weiden Ulmen Kastanien Buchen Erlen`,
	),
	stems:  strings.Fields(`Linden Birken Eichen Buchen Ahorn Kastanien Tannen Erlen Ulmen Rosen Tulpen Flieder Garten Schul Kirch Bahnhof Berg Wald Wiesen Feld Mühlen Brunnen Bach Schloss Post Dorf Haupt Sonnen Mozart Schiller Goethe Lessing Beethoven Uhland Kant Humboldt Schubert Haydn Brahms Dürer Kepler Gutenberg Fichten Kiefern Weiden Holunder Amsel Lerchen Finken Drossel Meisen`),
	places: parsePlaces(`Münster:481 Bielefeld:336 Osnabrück:490 Oldenburg:261 Göttingen:370 Kassel:341 Marburg:350 Gießen:353 Würzburg:970 Bamberg:960 Bayreuth:954 Regensburg:930 Passau:940 Ingolstadt:850 Augsburg:861 Ulm:890 Freiburg:791 Konstanz:784 Tübingen:720 Heilbronn:740 Heidelberg:691 Karlsruhe:761 Mainz:551 Trier:542 Koblenz:560 Saarbrücken:661 Erfurt:990 Jena:077 Weimar:994 Halle:061 Magdeburg:391 Potsdam:144 Rostock:180 Schwerin:190 Lübeck:235 Kiel:241 Flensburg:249 Paderborn:331 Lüneburg:213 Braunschweig:381`),
	alt:    parsePlaces(`Graz:8010 Linz:4020 Salzburg:5020 Innsbruck:6020 Klagenfurt:9020 Villach:9500 Wels:4600 Steyr:4400 Dornbirn:6850 Bregenz:6900 Leoben:8700 Krems:3500 Eisenstadt:7000 Feldkirch:6800 Bern:3011 Luzern:6003 Winterthur:8400 Thun:3600 Chur:7000 Schaffhausen:8200 Aarau:5000 Zug:6300 Olten:4600 Solothurn:4500 Frauenfeld:8500 Biel:2502 Wil:9500 Uster:8610`),
}

const addrAustrianPlaces = 14

var addrNL = &addrLocale{
	id:       "nl",
	given:    strings.Fields(`Jan Willem Pieter Hendrik Cornelis Johan Frederik Maurits Jacob Dirk Anna Maria Michiel Piet Joost Hugo`),
	family:   strings.Fields(`Rembrandt Vermeer Erasmus Vondel Spinoza Huygens Tasman Ruyter Tromp Hals Steen Mesdag Tollens Potter Evertsen Heemskerck`),
	cityLead: strings.Fields(`Nieuw Oud Sint Noord Zuid Groot Klein Hoog Laag`),
	cityTail: strings.Fields(`Rijn Maas IJssel Vecht Waal Amstel Lek Zee Zoom Linge Dommel Schie`),
	streets: slotLists(
		"", `Oranje Nassau Juliana Beatrix Wilhelmina Emma Irene Margriet Bernhard Rembrandt Vermeer Erasmus Vondel Spinoza Huygens Tasman Ruyter Tromp`,
	),
	stems:  strings.Fields(`Kerk Dorps Molen School Stations Linden Beuken Eiken Berken Wilgen Kastanje Rozen Tulpen Juliana Beatrix Wilhelmina Oranje Nieuw Hoog Zuider Noorder Ooster Wester Haven Vaart Brink Meer Bos Veld Heide Zand Duin Park Burg Kasteel Klooster Prinsen Koning Emma Irene Margriet Bernhard Meidoorn Esdoorn Populieren Iepen`),
	places: parsePlaces(`Zwolle:80 Deventer:74 Apeldoorn:73 Arnhem:68 Nijmegen:65 Enschede:75 Hengelo:75 Amersfoort:38 Hilversum:12 Haarlem:20 Leiden:23 Delft:26 Gouda:28 Dordrecht:33 Breda:48 Tilburg:50 Eindhoven:56 Helmond:57 Venlo:59 Roermond:60 Maastricht:62 Middelburg:43 Alkmaar:18 Hoorn:16 Leeuwarden:89 Groningen:97 Assen:94 Emmen:78 Zutphen:72 Doetinchem:70 Oss:53 Roosendaal:47 Heerlen:64 Zaandam:15 Almere:13 Lelystad:82`),
}

var addrUK = &addrLocale{
	id:       "uk",
	given:    strings.Fields(`Great Little Upper Lower North South East West Old New High Long`),
	cityLead: strings.Fields(`Great Little Market Bishop North South East West Upper Lower New Long Old Kings`),
	cityTail: strings.Fields(`Avon Thames Severn Trent Tyne Wye Ouse Dee Tees Humber Wear Exe Tamar Stour`),
	streets: slotLists(
		"", `Church Mill Station Victoria Albert Manor Chapel Queens Kings George York Windsor Richmond Springfield Highfield Westfield Northfield Southfield Orchard Rectory Vicarage School Bridge Castle Abbey Priory Cromwell Nelson Wellington Alexandra Clarence Salisbury Stanley Grosvenor Belmont Claremont Oakfield Ashfield Beechwood Elmfield Holly Cedar Willow Hawthorn Chestnut Sycamore Rowan Poplar`,
	),
	places: parsePlaces(`York:YO Exeter:EX Bath:BA Lincoln:LN Norwich:NR Ipswich:IP Chester:CH Carlisle:CA Durham:DH Lancaster:LA Preston:PR Shrewsbury:SY Hereford:HR Worcester:WR Gloucester:GL Swindon:SN Reading:RG Oxford:OX Cambridge:CB Peterborough:PE Northampton:NN Leicester:LE Derby:DE Nottingham:NG Doncaster:DN Harrogate:HG Darlington:DL Stirling:FK Perth:PH Inverness:IV Dundee:DD Aberdeen:AB Swansea:SA Cardiff:CF Newport:NP Plymouth:PL Truro:TR Taunton:TA Salisbury:SP Canterbury:CT Colchester:CO Luton:LU Stevenage:SG Guildford:GU Brighton:BN Birmingham:B Glasgow:G Liverpool:L Manchester:M Sheffield:S`),
}

var addrUKTypes = addrWordSet(`street st road rd lane ln close avenue ave drive dr way crescent court ct place pl terrace gardens garden grove square sq row walk mews hill green park view rise parade vale wynd`)

var addrGermanicWords = addrWordSet(`straße strasse str platz weg gasse ring damm ufer chaussee laan straat allee markt plein gracht singel dijk kade steeg dreef am an im zum zur auf unter`)

var addrConnWords = addrWordSet(`in op aan der den dem de la las los del des du le les el van di della delle dei dello degli do da dos das the sur sous en am an im upon on y e i lès lez o a os as il lo gli`)

var addrUnitWords = addrWordSet(`
flat apartment apt unit suite ste room floor bldg fl rm po box bis ter no num número nr
bâtiment bat appartement appt étage etage porte escalier
esc escalera piso planta puerta pta bloque portal izq izda dcha drcha bajo
andar esq dto dt frente fte lote
scala interno int piano
hinterhaus og eg stock wohnung whg top stiege tür
huis bus`)

var addrCountryWords = addrWordSet(`france deutschland germany españa espana spain italia italy portugal nederland netherlands österreich austria schweiz suisse switzerland uk england scotland wales belgique belgië united kingdom`)

var (
	addrUKPostEndRE   = regexp.MustCompile(`(?:^|[\s,])([A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2})(?:,?\s+(?i:UK|United Kingdom|England|Scotland|Wales|Northern Ireland|Great Britain))?$`)
	addrNLPostRE      = regexp.MustCompile(`\b\d{4} ?[A-Z]{2}\b`)
	addrPTPostRE      = regexp.MustCompile(`\b\d{4}-\d{3}\b`)
	addrFivePostRE    = regexp.MustCompile(`\b\d{5}\b`)
	addrFourPostRE    = regexp.MustCompile(`\b(\d{4}) \p{Lu}`)
	addrFRSureRE      = regexp.MustCompile(`(?:^|[ ,\n])\d{1,4}(?: ?(?:bis|ter))?,? (?i:rue|boulevard|bd\.?|chemin|allée|allee|impasse|quai|cours) \S`)
	addrFRMaybeRE     = regexp.MustCompile(`(?:^|[ ,\n])\d{1,4}(?: ?(?:bis|ter))?,? ((?i:avenue|av\.?|place|route|passage|square)) [\p{L}]`)
	addrEUTailRE      = regexp.MustCompile(`\b\d{5} \p{Lu}`)
	addrUSZipEndRE    = regexp.MustCompile(`[A-Za-z.],? {1,2}\d{5}(?:-\d{4})?\s*$`)
	addrUSStateEndRE  = regexp.MustCompile(`,\s*[A-Z]{2}\s*$`)
	addrLocElisionRE  = regexp.MustCompile(`^(?i:d|l|dell|all|nell|sull|dall)['’]\p{L}`)
	addrGermanLeadRE  = regexp.MustCompile(`^(?:Am|An|Im|Zum|Zur|Auf|Unter|In) `)
	addrDutchLeadRE   = regexp.MustCompile(`^(?:Op|Aan) `)
	addrDutchSuffixes = addrWordSet(`straat laan gracht plein singel dijk steeg dreef kade`)
	addrRomanRE       = regexp.MustCompile(`^[IVX]{2,5}$`)
	addrUSStateZipRE  = regexp.MustCompile(`\b[A-Z]{2},? {1,2}\d{5}(?:-\d{4})?\s*$`)
	addrUKCountryRE   = regexp.MustCompile(`(?:[,\n]\s*(?:UK|United Kingdom|England|Scotland|Wales)|[^,\n]\s+UK)\.?\s*$`)
	addrUKStreetRE    = regexp.MustCompile(`(?:^|[ ,\n])\d{1,4}[A-Za-z]? (?:\p{Lu}[\p{L}'’]* ){1,3}(?:Street|St|Road|Rd|Lane|Close|Avenue|Ave|Drive|Way|Crescent|Court|Place|Terrace|Gardens|Grove|Square|Row|Walk|Mews|Hill|Green|Park|View|Rise|Parade|Vale|Wynd)\b`)
	addrEnglishTails  = addrWordSet(`street st avenue ave road rd blvd lane ln drive dr court ct place pl way terrace ter circle cir parkway pkwy highway hwy square sq trail trl loop run pike n s e w ne nw se sw north south east west`)
)

const (
	addrUSStateNames = `Alabama|Alaska|Arizona|Arkansas|California|Colorado|Connecticut|Delaware|Florida|Georgia|Hawaii|Idaho|Illinois|Indiana|Iowa|Kansas|Kentucky|Louisiana|Maine|Maryland|Massachusetts|Michigan|Minnesota|Mississippi|Missouri|Montana|Nebraska|Nevada|New Hampshire|New Jersey|New Mexico|New York|North Carolina|North Dakota|Ohio|Oklahoma|Oregon|Pennsylvania|Rhode Island|South Carolina|South Dakota|Tennessee|Texas|Utah|Vermont|Virginia|Washington|West Virginia|Wisconsin|Wyoming|District of Columbia|Puerto Rico`
	addrUSStateCodes = `AL|AK|AZ|AR|CA|CO|CT|DE|FL|GA|HI|ID|IL|IN|IA|KS|KY|LA|ME|MD|MA|MI|MN|MS|MO|MT|NE|NV|NH|NJ|NM|NY|NC|ND|OH|OK|OR|PA|RI|SC|SD|TN|TX|UT|VT|VA|WA|WV|WI|WY|DC|PR`
	addrUSZipTail    = `(?:,?\s+\d{5}(?:-\d{4})?)?[\s.,]*$`
)

var (
	addrUSCountryEndRE = regexp.MustCompile(`(?:^|[,\s])(?:USA|U\.S\.A\.?|U\.S\.?|US|(?i:United\s+States(?:\s+of\s+America)?))[\s.]*$`)
	addrUSNameEndRE    = regexp.MustCompile(`(?i)(?:^|[,\s])(?:` + strings.ReplaceAll(addrUSStateNames, " ", `\s+`) + `)` + addrUSZipTail)
	addrUSCodeEndRE    = regexp.MustCompile(`[,\s](?:` + addrUSStateCodes + `)` + addrUSZipTail)
	addrUSAnyZipEndRE  = regexp.MustCompile(`[\p{L}.],?\s+\d{5}(?:-\d{4})?[\s.,]*$`)
)

var (
	addrPostFirstRE     = regexp.MustCompile(`^(?:[A-Z]{1,2}-)?(?:\d{5}|\d{4}-\d{3}|\d{4}(?: ?[A-Z]{2})?) \p{Lu}[\p{L}'’\-]*(?: [\p{L}'’\-]+){0,3}$`)
	addrPostFirstRestRE = regexp.MustCompile(`\b\d{5}\b|\b\d{4}-\d{3}\b|\b\d{4} ?[A-Z]{2}\b|(?:^|[ ,])[A-Z]{1,2}-\d{4}`)
)

type addrPostFirst struct {
	post, sep, rest string
	trailing        bool
}

func addrPostcodeFirst(real string) (addrPostFirst, bool) {
	c := strings.IndexByte(real, ',')
	if c < 0 || strings.ContainsAny(real, "\n") {
		return addrPostFirst{}, false
	}
	post := strings.TrimRight(real[:c], " \t")
	m := c + 1
	for m < len(real) && (real[m] == ' ' || real[m] == '\t') {
		m++
	}
	pf := addrPostFirst{post: post, sep: real[len(post):m], rest: real[m:]}
	if !addrPostFirstRE.MatchString(post) || !strings.ContainsAny(pf.rest, "0123456789") || addrUSMarked(pf.rest) {
		return addrPostFirst{}, false
	}
	if addrPostFirstRestRE.MatchString(pf.rest) {
		k := strings.LastIndex(pf.rest, ", ")
		if k < 0 || !addrPostFirstRE.MatchString(pf.rest[k+2:]) || addrPostFirstRestRE.MatchString(pf.rest[:k]) {
			return addrPostFirst{}, false
		}
		pf.trailing = true
	}
	return pf, true
}

func (pf addrPostFirst) canon() string {
	if pf.trailing {
		return pf.rest
	}
	return pf.rest + ", " + pf.post
}

var addrOddSpaceRE = regexp.MustCompile(`[ \t]+,|[ \t]{2,}|\t`)

func addrNormalSpacing(real string) string {
	return addrOddSpaceRE.ReplaceAllStringFunc(real, func(m string) string {
		if strings.HasSuffix(m, ",") {
			return ","
		}
		return " "
	})
}

func addrUSMarked(real string) bool {
	return addrUSAnyZipEndRE.MatchString(real) || addrUSCountryEndRE.MatchString(real) || addrUSNameEndRE.MatchString(real) || addrUSCodeEndRE.MatchString(real)
}

func addrLooseUK(real string) *addrLocale {
	if addrUSMarked(real) || addrUSStateZipRE.MatchString(real) || !addrUKStreetRE.MatchString(real) || !addrUKCountryRE.MatchString(real) {
		return nil
	}
	return addrUK
}

func addrRomanceLocale(first, third string, hasPT bool) *addrLocale {
	switch third {
	case "do", "da", "dos", "das":
		hasPT = hasPT || first == "avenida" || first == "av" || first == "largo"
	}
	switch {
	case first == "avenida" || first == "av":
		if hasPT {
			return addrPT
		}
		return addrES
	case first == "largo":
		if hasPT {
			return addrPT
		}
		return addrIT
	case addrPT.types[first]:
		return addrPT
	case addrES.types[first]:
		return addrES
	case addrIT.types[first]:
		return addrIT
	}
	return nil
}

func addrNumberFirstLocale(real string, toks [][]int) *addrLocale {
	text := func(i int) string {
		if i >= len(toks) {
			return ""
		}
		return real[toks[i][0]:toks[i][1]]
	}
	word := text(2)
	c, _ := utf8.DecodeRuneInString(word)
	if sep := text(1); sep != " " && sep != ", " || !unicode.IsLetter(c) || addrUSMarked(real) || addrUSStateZipRE.MatchString(real) || addrUSStateEndRE.MatchString(real) {
		return nil
	}
	rest := real[toks[0][1]:]
	hasNL := addrNLPostRE.MatchString(rest)
	hasPT := addrPTPostRE.MatchString(rest)
	euPost := addrEUTailRE.MatchString(rest) || hasPT
	if !euPost && !hasNL && !addrFourPostRE.MatchString(rest) {
		return nil
	}
	lineEnd := len(real)
	if k := strings.IndexAny(real, ",\n"); k >= 0 {
		lineEnd = k
	}
	tail := ""
	for i := 4; i < len(toks) && toks[i][1] <= lineEnd; i++ {
		if r0, _ := utf8.DecodeRuneInString(text(i)); unicode.IsLetter(r0) {
			tail = strings.ToLower(text(i))
		}
	}
	if addrEnglishTails[tail] {
		return nil
	}
	low := strings.ToLower(word)
	if loc := addrRomanceLocale(low, strings.ToLower(text(4)), hasPT); loc != nil {
		if toks[0][1]-toks[0][0] > 3 && !euPost {
			return nil
		}
		return loc
	}
	if !unicode.IsUpper(c) {
		return nil
	}
	suffix := ""
	if m := addrAttachedRE.FindStringSubmatch(word); m != nil {
		suffix = strings.ToLower(m[1])
	} else if next := strings.ToLower(text(4)); text(3) == " " && addrGermanicWords[next] && unicode.IsUpper([]rune(text(4))[0]) && !addrGermanicWords[low] {
		suffix = next
	}
	switch {
	case suffix == "":
		return nil
	case addrDutchSuffixes[suffix] || hasNL:
		return addrNL
	}
	return addrDE
}

func addrLocaleOf(real string) *addrLocale {
	if real == "" {
		return nil
	}
	if pf, ok := addrPostcodeFirst(real); ok {
		if loc := addrLocaleOf(pf.canon()); loc != nil && loc != addrUK {
			return loc
		}
	}
	if addrUKPostEndRE.MatchString(real) {
		return addrUK
	}
	usTail := addrUSZipEndRE.MatchString(real) || addrUSStateEndRE.MatchString(real) && !addrEUTailRE.MatchString(real)
	if !usTail {
		if addrFRSureRE.MatchString(real) {
			return addrFR
		}
		if m := addrFRMaybeRE.FindStringSubmatch(real); m != nil && (m[1] == strings.ToLower(m[1]) || addrEUTailRE.MatchString(real)) {
			return addrFR
		}
	}
	toks := addrTokenRE.FindAllStringIndex(real, -1)
	if real[0] >= '0' && real[0] <= '9' {
		if loc := addrNumberFirstLocale(real, toks); loc != nil {
			return loc
		}
		return addrLooseUK(real)
	}
	first := strings.ToLower(real[toks[0][0]:toks[0][1]])
	hasNL := addrNLPostRE.MatchString(real)
	third := ""
	if len(toks) > 2 {
		third = strings.ToLower(real[toks[2][0]:toks[2][1]])
	}
	if loc := addrRomanceLocale(first, third, addrPTPostRE.MatchString(real)); loc != nil {
		return loc
	}
	for i, t := range toks {
		if i+2 >= len(toks) {
			break
		}
		tok := real[t[0]:t[1]]
		r0, _ := utf8.DecodeRuneInString(tok)
		if !unicode.IsUpper(r0) {
			continue
		}
		sep := real[toks[i+1][0]:toks[i+1][1]]
		if sep != " " && sep != ". " || real[toks[i+2][0]] < '0' || real[toks[i+2][0]] > '9' {
			continue
		}
		low := strings.ToLower(tok)
		suffix := ""
		if addrGermanicWords[low] {
			suffix = low
		} else if m := addrAttachedRE.FindStringSubmatch(tok); m != nil {
			suffix = strings.ToLower(m[1])
		}
		switch {
		case suffix == "" || suffix == "am" || suffix == "an" || suffix == "im" || suffix == "zum" || suffix == "zur" || suffix == "auf" || suffix == "unter":
			continue
		case addrDutchSuffixes[suffix]:
			return addrNL
		case hasNL:
			return addrNL
		}
		return addrDE
	}
	if addrGermanLeadRE.MatchString(real) && (addrFivePostRE.MatchString(real) || addrFourPostRE.MatchString(real)) && !hasNL {
		return addrDE
	}
	if hasNL && (addrDutchLeadRE.MatchString(real) || addrGermanLeadRE.MatchString(real)) {
		return addrNL
	}
	return addrLooseUK(real)
}

func (l *addrLocale) postalSpan(real string) (int, int) {
	last := func(ms [][]int, a, b int) (int, int) {
		for i := len(ms) - 1; i >= 0; i-- {
			if ms[i][a] >= 0 {
				return ms[i][a], ms[i][b]
			}
		}
		return -1, -1
	}
	switch l.id {
	case "uk":
		if m := addrUKPostEndRE.FindStringSubmatchIndex(real); m != nil {
			return m[2], m[3]
		}
		return -1, -1
	case "nl":
		return last(addrNLPostRE.FindAllStringIndex(real, -1), 0, 1)
	case "pt":
		return last(addrPTPostRE.FindAllStringIndex(real, -1), 0, 1)
	case "de":
		if s, e := last(addrFivePostRE.FindAllStringIndex(real, -1), 0, 1); s >= 0 {
			return s, e
		}
		return last(addrFourPostRE.FindAllStringSubmatchIndex(real, -1), 2, 3)
	}
	return last(addrFivePostRE.FindAllStringIndex(real, -1), 0, 1)
}

const (
	addrNLLetters = "ABCDEGHJKLMNPRSTVWXZ"
	addrUKLetters = "ABDEFGHJLNPQRSTUWXYZ"
)

func (l *addrLocale) postal(r *rng, realPost string, p addrPlace) string {
	var out string
	switch l.id {
	case "es":
		out = p.code + "00" + string(rune('1'+r.intn(9)))
	case "de":
		out = p.code
		if len(out) == 3 {
			out += r.digits(2, false)
		}
	case "pt":
		out = p.code + "-" + r.digits(3, false)
	case "nl":
		sep := ""
		if strings.Contains(realPost, " ") {
			sep = " "
		}
		out = p.code + r.digits(2, false) + sep + string(addrNLLetters[r.intn(len(addrNLLetters))]) + string(addrNLLetters[r.intn(len(addrNLLetters))])
	case "uk":
		area := 0
		for area < len(realPost) && realPost[area] >= 'A' && realPost[area] <= 'Z' {
			area++
		}
		outward := strings.TrimSpace(realPost[area : len(realPost)-3])
		sep := ""
		if strings.Contains(realPost, " ") {
			sep = " "
		}
		district := string(rune('1' + r.intn(9)))
		if len(outward) > 1 {
			district = "1" + r.digits(1, false)
		}
		out = p.code + district + sep + r.digits(1, false) + string(addrUKLetters[r.intn(len(addrUKLetters))]) + string(addrUKLetters[r.intn(len(addrUKLetters))])
	default:
		out = p.code
	}
	if out == realPost {
		last := out[len(out)-1]
		if last >= '0' && last <= '9' {
			out = out[:len(out)-1] + string(rune('0'+(last-'0'+1)%10))
		}
	}
	return out
}

func (r *rng) pickAvoid(list []string, avoid map[string]bool) string {
	start := r.intn(len(list))
	for k := range list {
		c := list[(start+k)%len(list)]
		if !avoid[textutil.Fold(c)] {
			return c
		}
	}
	return list[start]
}

func addrVowelStart(s string) bool {
	f := textutil.Fold(s)
	return f != "" && strings.IndexByte("aeiou", f[0]) >= 0
}

func (r *rng) pickPlace(list []addrPlace, avoid map[string]bool, codeLen int, vowel bool) addrPlace {
	start := r.intn(len(list))
	for pass := 0; pass < 2; pass++ {
		for k := range list {
			p := list[(start+k)%len(list)]
			if !avoid[textutil.Fold(p.name)] && (codeLen == 0 || len(p.code) == codeLen) && (!vowel || pass > 0 || addrVowelStart(p.name)) {
				return p
			}
		}
	}
	return list[start]
}

func (g *Generator) localAddress(r *rng, real string, loc *addrLocale) string {
	if pf, ok := addrPostcodeFirst(real); ok && addrLocaleOf(pf.canon()) == loc {
		out := g.localAddress(r, pf.canon(), loc)
		k := strings.LastIndex(out, ", ")
		if out == pf.canon() || k < 0 {
			return real
		}
		if !pf.trailing {
			return out[k+2:] + pf.sep + out[:k]
		}
		street := pf.rest[:strings.LastIndex(pf.rest, ", ")]
		lead := g.localAddress(r, street+", "+pf.post, loc)
		l := strings.LastIndex(lead, ", ")
		if l < 0 || lead == street+", "+pf.post {
			return real
		}
		return lead[l+2:] + pf.sep + out
	}
	toks := addrTokenRE.FindAllStringIndex(real, -1)
	n := len(toks)
	text := func(i int) string { return real[toks[i][0]:toks[i][1]] }
	first := func(i int) rune {
		c, _ := utf8.DecodeRuneInString(real[toks[i][0]:])
		return c
	}
	isWord := func(i int) bool {
		if i < 0 || i >= n {
			return false
		}
		c := first(i)
		return unicode.IsLetter(c) || c == '\'' || c == '’'
	}
	isDigits := func(i int) bool { return i >= 0 && i < n && unicode.IsDigit(first(i)) }

	avoid := map[string]bool{}
	for i := range toks {
		if !isWord(i) {
			continue
		}
		tok := text(i)
		avoid[textutil.Fold(tok)] = true
		if k := strings.IndexAny(tok, "'’"); k >= 0 {
			_, size := utf8.DecodeRuneInString(tok[k:])
			avoid[textutil.Fold(tok[k+size:])] = true
		}
		if m := addrAttachedRE.FindStringSubmatch(tok); m != nil {
			avoid[textutil.Fold(tok[:len(tok)-len(m[1])])] = true
		}
	}

	ps, pe := loc.postalSpan(real)
	pIdx := -1
	for i, t := range toks {
		if t[0] == ps {
			pIdx = i
		}
	}
	if pIdx < 0 {
		ps, pe = -1, -1
	}

	typeTok := -1
	switch {
	case loc.typeFirst:
		for i := range toks {
			if isWord(i) && loc.types[strings.ToLower(text(i))] {
				typeTok = i
				break
			}
		}
	case loc.id == "uk":
		house := -1
		for i := range toks {
			if pIdx >= 0 && i >= pIdx {
				break
			}
			if isDigits(i) && i+2 < n && text(i+1) == " " && isWord(i+2) {
				house = i
				break
			}
		}
		for i := house + 4; house >= 0 && i < n && (pIdx < 0 || i < pIdx); i += 2 {
			if text(i-1) != " " || !isWord(i) {
				break
			}
			if addrUKTypes[strings.ToLower(text(i))] {
				typeTok = i
			} else if typeTok >= 0 {
				break
			}
		}
	}

	primaryTok := -1
	if pIdx >= 0 {
		q := pIdx
		for q+1 < n && toks[q+1][0] < pe {
			q++
		}
		if q+2 < n && text(q+1) == " " && isWord(q+2) {
			primaryTok = q + 2
		} else if pIdx >= 2 && !isWord(pIdx-1) && !isDigits(pIdx-1) && isWord(pIdx-2) && pIdx-2 != typeTok && (loc.id == "uk" || text(pIdx-1) == " ") {
			j := pIdx - 2
			for j >= 2 && (text(j-1) == " " || text(j-1) == "-") && isWord(j-2) && j-2 != typeTok {
				j -= 2
			}
			primaryTok = j
		}
	}

	places := loc.places
	codeLen := 0
	switch {
	case loc.id == "de" && pIdx >= 0 && pe-ps == 4:
		places = loc.alt
		switch {
		case ps >= 3 && real[ps-3:ps] == "CH-":
			places = loc.alt[addrAustrianPlaces:]
		case strings.Contains(real, "ß") || ps >= 2 && real[ps-2:ps] == "A-":
			places = loc.alt[:addrAustrianPlaces]
		}
	case loc.id == "uk" && pIdx >= 0:
		for codeLen < pe-ps && real[ps+codeLen] >= 'A' && real[ps+codeLen] <= 'Z' {
			codeLen++
		}
	}
	place := r.pickPlace(places, avoid, codeLen, primaryTok >= 0 && addrLocElisionRE.MatchString(text(primaryTok)))
	avoid[textutil.Fold(place.name)] = true

	germanic := loc.id == "de" || loc.id == "nl"
	seenStreet, seenNumber, city, usedPrimary := false, false, false, false
	slot := ""

	isName := func(k int) bool {
		if !isWord(k) || k == primaryTok || k == typeTok || toks[k][0] == ps {
			return false
		}
		tok := text(k)
		low := strings.ToLower(tok)
		if utf8.RuneCountInString(tok) < 2 || city && addrCountryWords[low] || addrUnitWords[low] || addrConnWords[low] {
			return false
		}
		if germanic && !city && (addrGermanicWords[low] || addrAttachedRE.MatchString(tok)) {
			return false
		}
		return unicode.IsUpper(first(k)) || addrLocElisionRE.MatchString(tok)
	}
	runEnd := func(i int) int {
		j := i
		for {
			k := j + 1
			if k >= n || text(k) != " " && text(k) != "-" {
				return j
			}
			k++
			for k+1 < n && isWord(k) && addrConnWords[strings.ToLower(text(k))] && (text(k+1) == " " || text(k+1) == "-") {
				k += 2
			}
			if !isName(k) {
				return j
			}
			j = k
		}
	}
	var b strings.Builder
	pickWord := func(src string, list []string) string {
		w := r.pickAvoid(list, avoid)
		avoid[textutil.Fold(w)] = true
		return matchWordCase(src, w)
	}
	streetList := func(key string) []string {
		list := loc.streets[key]
		if list == nil {
			list = loc.streets[""]
		}
		if list == nil {
			list = loc.streets["de"]
		}
		return list
	}
	elision := func(tok string) (string, string) {
		if !addrLocElisionRE.MatchString(tok) {
			return "", tok
		}
		k := strings.IndexAny(tok, "'’")
		_, size := utf8.DecodeRuneInString(tok[k:])
		return tok[:k+size], tok[k+size:]
	}
	const (
		runKeep = iota
		runConn
		runElided
		runName
	)
	streetClass := func(k int, head bool) int {
		tok := text(k)
		low := strings.ToLower(tok)
		switch {
		case addrLocElisionRE.MatchString(tok):
			return runElided
		case head:
			return runName
		case addrRomanRE.MatchString(tok) || loc.id == "uk" && addrUKTypes[low]:
			return runKeep
		case addrConnWords[low]:
			return runConn
		}
		return runName
	}
	streetRun := func(i int) int {
		end := runEnd(i)
		var groups []int
		open := false
		for k := i; k <= end; k++ {
			if !isWord(k) {
				continue
			}
			switch streetClass(k, k == i) {
			case runName:
				if !open {
					groups = append(groups, 0)
					open = true
				}
				groups[len(groups)-1]++
			case runConn, runElided:
				open = false
			}
		}
		key := slot
		gi, pi := 0, 0
		for k := i; k <= end; k++ {
			tok := text(k)
			if !isWord(k) {
				b.WriteString(tok)
				continue
			}
			switch streetClass(k, k == i) {
			case runKeep:
				b.WriteString(tok)
			case runConn:
				b.WriteString(tok)
				key = strings.ToLower(tok)
			case runElided:
				prefix, rest := elision(tok)
				key = "l'"
				if strings.EqualFold(strings.TrimRight(prefix, "'’"), "d") {
					key = "d'"
				}
				b.WriteString(prefix + pickWord(rest, streetList(key)))
				key = ""
			default:
				size := groups[gi]
				var list []string
				switch {
				case size == 1:
					list = streetList(key)
					if germanic && loc.family != nil && k+1 < n && text(k+1) == "-" {
						list = loc.family
					}
				case pi < size-1:
					list = loc.given
					if pi == 0 && loc.titleSlot[key] {
						list = loc.titles
					}
				default:
					list = loc.family
				}
				if list == nil {
					list = streetList("")
				}
				b.WriteString(pickWord(tok, list))
				key = ""
				if pi++; pi == size {
					gi, pi = gi+1, 0
				}
			}
		}
		slot = ""
		seenStreet = true
		return end
	}
	cityRun := func(i int) int {
		end := runEnd(i)
		usePlace := i == primaryTok || primaryTok < 0 && !usedPrimary
		if usePlace {
			usedPrimary = true
		}
		heads := 0
		for k := i; k <= end; k++ {
			if !isWord(k) {
				continue
			}
			if addrConnWords[strings.ToLower(text(k))] {
				if heads > 0 {
					break
				}
				continue
			}
			if heads > 0 && addrLocElisionRE.MatchString(text(k)) {
				break
			}
			heads++
		}
		seen, interior, key := 0, false, ""
		for k := i; k <= end; k++ {
			tok := text(k)
			if !isWord(k) {
				b.WriteString(tok)
				continue
			}
			low := strings.ToLower(tok)
			prefix, rest := elision(tok)
			switch {
			case addrConnWords[low]:
				b.WriteString(tok)
				interior = interior || seen > 0
				key = low
			case interior || prefix != "" && seen > 0:
				interior = true
				list := loc.cityTail
				if art := loc.streets[key]; loc.typeFirst && art != nil && prefix == "" && key != "" && key != "de" && key != "di" {
					list = art
				}
				if prefix != "" {
					var vowels []string
					for _, w := range list {
						if addrVowelStart(w) {
							vowels = append(vowels, w)
						}
					}
					if vowels != nil {
						list = vowels
					}
				}
				b.WriteString(prefix + pickWord(rest, list))
			case seen < heads-1:
				b.WriteString(pickWord(tok, loc.cityLead))
				seen++
			default:
				name := place.name
				if !usePlace {
					name = r.pickPlace(places, avoid, 0, prefix != "").name
				}
				b.WriteString(prefix + matchWordCase(rest, name))
				seen++
			}
		}
		return end
	}

	repeat := -1
	if pIdx >= 2 && text(pIdx-1) == " " && text(pIdx-2) == real[ps:pe] {
		repeat = pIdx - 2
	}
	for i := 0; i < n; i++ {
		tok := text(i)
		if repeat >= 0 && (i == repeat || i == repeat+1) {
			continue
		}
		if i == pIdx {
			pc := loc.postal(r, real[ps:pe], place)
			if repeat >= 0 {
				b.WriteString(pc + " ")
			}
			b.WriteString(pc)
			for i+1 < n && toks[i+1][0] < pe {
				i++
			}
			city = true
			continue
		}
		if i == primaryTok {
			city = true
		}
		c := first(i)
		switch {
		case unicode.IsDigit(c):
			if len(tok) >= 4 && seenStreet && seenNumber {
				city = true
			}
			d := r.digits(len(tok), tok[0] != '0' && len(tok) < 5)
			for k := 0; d == tok && k < 16; k++ {
				d = r.digits(len(tok), tok[0] != '0' && len(tok) < 5)
			}
			b.WriteString(d)
			seenNumber = true
			slot = ""
		case !isWord(i):
			if seenStreet && seenNumber && strings.ContainsAny(tok, ",\n") {
				city = true
			}
			b.WriteString(tok)
		default:
			low := strings.ToLower(tok)
			single := utf8.RuneCountInString(tok) == 1
			switch {
			case pIdx >= 0 && toks[i][1] == ps-1 && real[ps-1] == '-' && len(tok) <= 2:
				b.WriteString(tok)
			case i == typeTok:
				seenStreet = true
				slot = ""
				b.WriteString(tok)
				if loc.id == "uk" {
					city = true
				}
			case city && addrCountryWords[low] && !single:
				b.WriteString(tok)
			case germanic && !city && addrGermanicWords[low]:
				seenStreet = true
				slot = low
				b.WriteString(tok)
			case city && addrConnWords[low] && !single && i+2 < n && text(i+1) == " " && isName(i+2):
				i = cityRun(i)
			case addrConnWords[low] && !single || addrUnitWords[low]:
				slot = low
				b.WriteString(tok)
			case single:
				prevDigit := i > 0 && toks[i-1][1] == toks[i][0] && real[toks[i][0]-1] >= '0' && real[toks[i][0]-1] <= '9'
				if prevDigit || seenNumber && c < utf8.RuneSelf && unicode.IsUpper(c) {
					b.WriteString(shapeLike(r, tok))
				} else {
					b.WriteString(tok)
				}
			case addrLocElisionRE.MatchString(tok):
				if city {
					i = cityRun(i)
				} else {
					i = streetRun(i)
				}
			case !unicode.IsUpper(c) || addrRomanRE.MatchString(tok):
				b.WriteString(tok)
			case city:
				i = cityRun(i)
			default:
				if m := addrAttachedRE.FindStringSubmatch(tok); m != nil && germanic {
					suffix := strings.ToLower(m[1])
					stem := loc.stems[r.intn(len(loc.stems))]
					for k := 0; k < len(loc.stems) && (avoid[textutil.Fold(stem)] || avoid[textutil.Fold(stem+suffix)] || strings.ToLower(stem) == suffix); k++ {
						stem = loc.stems[r.intn(len(loc.stems))]
					}
					seenStreet = true
					slot = ""
					b.WriteString(matchWordCase(tok, stem+suffix))
				} else {
					i = streetRun(i)
				}
			}
		}
	}
	return b.String()
}
