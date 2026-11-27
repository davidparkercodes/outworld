package detect

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

var familyFirstStrong = wordSet(`nguyen tran pham hoang huynh phan dang bui ngo duong wang zhang liu chen huang zhao zhou xu zhu guo gao luo choi jung kang cho yoon jang lim shin seo kwon`)

var familyFirstWeak = wordSet(`le do ho he lin lee kim park sun ma han vo vu wu hu li yang ly long`)

var koreanFamilyNames = wordSet(`kim lee yi park pak choi choe jung jeong chung kang cho jo yoon yun jang chang lim im han oh seo suh shin sin kwon hwang ahn an song yoo yu ryu hong jeon jun ko go moon mun yang son bae baek paik heo hur nam noh roh ha kwak sung cha joo ju woo min na jin chae won cheon bang gong hyun ham byun yeom yeo chu seok sun gil yeon pyo myung ban ra geum ok yook maeng tak guk`)

var givenMarkers = wordSet(`van thi duc minh huu ngoc xuan quoc dinh cong nhu`)

var syllableWords = wordSet(`a an i in u y do so he ha can ran run die lie tie pie pen hen you gun gang bang pin pan fan man men sun son`)

var syllablePairWords = wordSet(`human china change panda woman women linen pagan sedan sudan yemen taiwan bali mali lima guru sushi data mini mama papa nana dada gaga haha`)

var givenSyllables = wordSet(`a ai an ang ao ba bai ban bang bao bei ben beng bi bian biao bie bin bing bo bu cai can cang cao ce cen ceng cha chai chan chang chao che chen cheng chi chong chou chu chuan chuang chui chun chuo ci cong cui cun cuo da dai dan dang dao de deng di dian diao die ding diu dong dou du duan dui dun duo en er fa fan fang fei fen feng fo fu gai gan gang gao ge gen geng gong gou gu gua guai guan guang gui gun guo ha hai han hang hao he hei hen heng hong hou hu hua huai huan huang hui hun huo ji jia jian jiang jiao jie jin jing jiong jiu ju juan jue jun ka kai kan kang kao ke ken keng kong kou ku kuai kuan kuang kui kun kuo la lai lan lang lao le lei leng li lian liang liao lie lin ling liu long lou lu luan lun luo ma mai man mang mao mei men meng mi mian miao min ming mo mou mu na nai nan nang nao nei neng ni nian niang niao nie ning niu nong nu nuan nuo ou pan pang pei peng pi pian piao pin ping po pu qi qian qiang qiao qin qing qiong qiu qu quan que qun ran rang rao ren reng rong rou ru ruan rui run ruo sa sai san sang se sen sha shan shang shao she shen sheng shi shou shu shuai shuang shui shun shuo si song su suan sui sun suo ta tai tan tang tao te teng ti tian tiao tie ting tong tou tu tuan tun tuo wa wan wang wei wen weng wo wu xi xia xian xiang xiao xie xin xing xiong xiu xu xuan xue xun ya yan yang yao ye yi yin ying yong you yu yuan yue yun zai zan ze zeng zha zhan zhang zhao zhe zhen zheng zhi zhong zhou zhu zhuang zhuo zi zong zou zu zuo
ae ah bae baek beom bok bom bong bum byeong byung chae cheol chul dae deok duk eun geon geun gi gil gwang gyeong gyu hae hee heon ho hoon hwa hwan hye hyeok hyeon hyo hyuk hyun il ja jae jeong jong joo joon kook kwan kwang kyeong kyo kyu kyun kyung moon mun myeong myung nam ok ra rae ri rim rin ro ryeong seo seok seon seong seul seung sik sil soo sook soon suk sung tae uk un won woo wook woon yeo yeol yeon yeong yoo yoon young yul
anh bich binh cam chau chien chinh cuc cuong dat diem diep dieu dinh doan duc dung duong duy gia giang hanh hau hien hiep hieu hoa hoai hoan hoang hue hung huong huu huy huyen khai khang khanh khiem khoa khoi khue kien kieu kim ky lam lien lieu linh loan loc loi luc luong ly manh minh my nga ngan nghi nghia ngoc nguyen nguyet nhan nhat nhi nhu nhung ninh oanh phat phi phong phu phuc phung phuong quang quoc quy quyen quynh sinh suong tam thach thai tham thang thanh thao thi thien thinh tho thoa thong thu thuan thuy thuong tien tin tinh toan tra tram trang tri trieu trinh trong truc trung truong tue tung tuyen tuyet uyen van vi viet vinh vu vuong vy yen`)

var englishSyllables = wordSet(`long young moon song soon hang ban bin van tan tin loan won sang rang sung hung ping bum dung yen tram rim ding duo fang gong la mi ti pi mu nu ho ah fa ye`)

var englishSyllablePairs = wordSet(`ace ache adieu age aha aide aiding ail ailing aiming ale alien along amen anger angle angling ani ante anti aping ashen ate auk axing aye baa bade bail bailing baiting baize bake bale baling banding bangle bani banning bantam banyan base basil bate bating baying bayou beige bending bide biding bier bike bile binding binge binning bite biting boa bode boding boil bola bole bombing boning book bosun boxing bumming bumping buying camping cancan canning canting cede ceding chaining chaise chance chanting chase cheer chichi chide chiding chile chili chiming chitin chunky chute cite citing conga cunning dacha daily dale dance danger dangle dangling date dating daze dean deer deli demo dice dieting dike dining divan dongle douche douse ducting dude duding duke duly dunce dungeon dunning duping ending envy era face fade fading fail fake fanning fate fating faun faxing faze fence fending foci foil fora foxing fuming fuse fuze gaily gaining gangling gangly genie gentian gibing gilding gouge guide guiding guile guise gunman gunmen gunning guying haiku hail hailing hake hale haling handing hanger hangman hangmen hanky hate hating hauling haying haze heeding heeling hence henna hero hexing hoaxing hobo hogan hole holing holy homing homy honing hook hoping hose house huge hula hunger hunting hyena jade jading jail japan japing jibing jingle jingling jinni jinxing junky junta jute kale kaolin keen kenning keying lace lade lading lake lama lambing lamer laming lamming lance landing lanky lase late laying laze lea lean leer levy lice lichen liege like liken lily liming limo limy liner linger lining linting lira lite loaner longer louse lubing lucky lunge lupin lute mace made mail mailing maiming maize make male manga mange manger mangle mangling manly manna manning manse mantra mate mating maxing maze mending mice mien mike mile milieu miming mince minding miner mingle mingling mining minting mite mixing moan mocha mode mole moping mote mouse mule muse mute muting myna nail nailing naming navy nice niche niece nite nixing nuance nude nuke okra outing panning pantie panting pier pigeon pike pile piling pining pinky pinning piping pique pita pixie poke poky pole poling pose puke puma qua quanta queen queer race rage rail raja rake range ranger ranting raping rate rating raze rending renting rice ride riding rile riling riming rimming rinse rise risen rite roan robin robing rode roil role roman rook roping rose rote rouge rouse route routing rude ruining rule ruling running ruse sage sail sailing sake sale sanding saner sari sarong sate satin sating saying sea seen seer semen semi sending senna sense sera sexing shade shading shake shaken shaky shale shaman shaming shanghai shantung shaping shebang sheen sheer shining shouting shunning shunting side siding silken silky silly silting silvan simian siren site siting situ size sooner suite suiting sumo sundae sunken sunning suntan tabu tail tailing take taken tale tamer taming tamping tangle tangling tanning taping taro taxi taxing tea teen tide tiding tier tike tile tiling timing tinge tingle tingling tingly tinning tinting tipi tiro titan toucan touting trace trade trading trail tramming tramping trice trike trite tuba tubing tuna tuner tuning tutu unman unpin unsung untie vanning via vice vigil vile visa vise wade wadi wading wage wail wake waken wale waling wangle wangling waning wanly wanna wanting wavy waxing wending woke woken wonky wooding wooer woolie wooly yea yeah yeoman yeomen yeti younger yule zebu zero zeta`)

var familyPreps = wordSet(`with from cc by to`)

var listSeps = map[string]bool{", ": true, " and ": true, ", and ": true, " & ": true, " or ": true, ", or ": true}

func familyContext(text string, s, e int, fam, given string) bool {
	lg := textutil.Fold(given)
	if englishSyllablePairs[lg] {
		return false
	}
	after := personVerbAfter(text, e)
	if after && fam != "he" && fam != "do" {
		return true
	}
	if personDirectedBefore(text, s) {
		return true
	}
	if englishSyllables[lg] {
		return false
	}
	rest := text[e:]
	if w := wordsBefore(text, s, 1); len(w) > 0 && familyPreps[w[0]] && (after || strings.HasPrefix(rest, ",")) {
		return true
	}
	for _, suf := range []string{"'s ", "’s "} {
		if strings.HasPrefix(rest, suf) {
			r, _ := utf8.DecodeRuneInString(rest[len(suf):])
			return unicode.IsLower(r)
		}
	}
	return false
}

func familyListed(text string, s, e int, given string, starts, ends map[int]bool) bool {
	lg := textutil.Fold(given)
	if englishSyllablePairs[lg] || englishSyllables[lg] {
		return false
	}
	for sep := range listSeps {
		if n := len(sep); (s >= n && text[s-n:s] == sep && ends[s-n]) || (strings.HasPrefix(text[e:], sep) && starts[e+n]) {
			return true
		}
	}
	return false
}

func syllableCount(w string) int {
	if givenSyllables[w] {
		return 1
	}
	for i := 1; i < len(w); i++ {
		if givenSyllables[w[:i]] && givenSyllables[w[i:]] {
			return 2
		}
	}
	return 0
}

func EastSyllables(w string) int { return syllableCount(textutil.Fold(w)) }

func eastGiven(w string) (ok, distinctive bool) {
	lw := textutil.Fold(w)
	if strings.ContainsFunc(lw, textutil.IsNameJoiner) {
		parts := textutil.NameJoinerFields(lw)
		if len(parts) != 2 || syllableCount(parts[0]) != 1 || syllableCount(parts[1]) != 1 || joinedWordBlocked(parts[0]+parts[1]) {
			return false, false
		}
		return true, true
	}
	switch syllableCount(lw) {
	case 1:
		return true, false
	case 2:
		return true, len(lw) >= 5 && !syllablePairWords[lw] && !englishSyllablePairs[lw]
	}
	return false, false
}

func familyFirst(text string, toks []nameTok, i int) (end, n int, canonical string, pending bool) {
	t := toks[i]
	if t.shape != shapeTitle {
		return 0, 0, "", false
	}
	fam := textutil.Fold(t.text)
	strong := familyFirstStrong[fam]
	if !strong && !familyFirstWeak[fam] && !koreanFamilyNames[fam] {
		return 0, 0, "", false
	}
	var given []string
	marker, plain, distinct := false, false, false
	j := i + 1
	for j < len(toks) && len(given) < 3 {
		g := toks[j]
		if g.shape != shapeTitle || !oneSpace(text, toks[j-1].end, g.start) || embedded(text, g.start, g.end) {
			break
		}
		word := g.text
		for _, suf := range []string{"'s", "’s"} {
			word = strings.TrimSuffix(word, suf)
		}
		lg := textutil.Fold(word)
		ok, d := eastGiven(word)
		if !ok || syllablePairWords[lg] || (len(given) == 2 && !givenMarkers[textutil.Fold(given[0])]) {
			break
		}
		if d {
			distinct = true
		}
		if givenMarkers[lg] {
			marker = true
		} else if !syllableWords[lg] && !stopCap[lg] {
			plain = true
		}
		given = append(given, word)
		j++
		if word != g.text {
			break
		}
	}
	for len(given) > 0 {
		lg := textutil.Fold(given[len(given)-1])
		if !syllableWords[lg] && !stopCap[lg] {
			break
		}
		if len(given) >= 2 && givenMarkers[textutil.Fold(given[len(given)-2])] && lg == "an" {
			break
		}
		given = given[:len(given)-1]
		j--
	}
	if len(given) == 0 {
		return 0, 0, "", false
	}
	end = toks[j-1].start + len(given[len(given)-1])
	if placeUse(text, t.start, end) {
		return 0, 0, "", false
	}
	accept := false
	switch {
	case strong:
		accept = plain || marker
		if len(given) == 1 && utf8.RuneCountInString(given[0]) < 2 {
			accept = false
		}
	case distinct:
		accept = true
	case len(given) >= 2:
		accept = marker || (fam != "he" && fam != "do" && plain && allPlain(given))
	default:
		plain = plain && !englishSyllablePairs[textutil.Fold(given[0])]
		accept = plain && (lowerNameContext(text, t.start, end) || familyContext(text, t.start, end, fam, given[0]))
		pending = plain && !accept && fam != "he" && fam != "do"
	}
	if !accept && !pending {
		return 0, 0, "", false
	}
	return end, j - 1 - i, strings.Join(given, " ") + " " + t.text, pending
}

func allPlain(given []string) bool {
	for _, g := range given {
		lg := textutil.Fold(g)
		if syllableWords[lg] || stopCap[lg] {
			return false
		}
	}
	return true
}
