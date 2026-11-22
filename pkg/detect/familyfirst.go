package detect

import (
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

var familyFirstStrong = wordSet(`nguyen tran pham hoang huynh phan dang bui ngo duong wang zhang liu chen huang zhao zhou xu zhu guo gao luo choi jung kang cho yoon jang lim shin seo kwon`)

var familyFirstWeak = wordSet(`le do ho he lin lee kim park sun ma han vo vu wu hu li yang ly long`)

var givenMarkers = wordSet(`van thi duc minh huu ngoc xuan quoc dinh cong nhu`)

var syllableWords = wordSet(`a an i in u y do so he ha can ran run die lie tie pie pen hen you gun gang bang pin pan fan man men sun son`)

var syllablePairWords = wordSet(`human china change panda woman women linen pagan sedan sudan yemen taiwan bali mali lima guru sushi data mini mama papa nana dada gaga haha`)

var givenSyllables = wordSet(`a ai an ang ao ba bai ban bang bao bei ben beng bi bian biao bie bin bing bo bu cai can cang cao ce cen ceng cha chai chan chang chao che chen cheng chi chong chou chu chuan chuang chui chun chuo ci cong cui cun cuo da dai dan dang dao de deng di dian diao die ding diu dong dou du duan dui dun duo en er fa fan fang fei fen feng fo fu gai gan gang gao ge gen geng gong gou gu gua guai guan guang gui gun guo ha hai han hang hao he hei hen heng hong hou hu hua huai huan huang hui hun huo ji jia jian jiang jiao jie jin jing jiong jiu ju juan jue jun ka kai kan kang kao ke ken keng kong kou ku kuai kuan kuang kui kun kuo la lai lan lang lao le lei leng li lian liang liao lie lin ling liu long lou lu luan lun luo ma mai man mang mao mei men meng mi mian miao min ming mo mou mu na nai nan nang nao nei neng ni nian niang niao nie ning niu nong nu nuan nuo ou pan pang pei peng pi pian piao pin ping po pu qi qian qiang qiao qin qing qiong qiu qu quan que qun ran rang rao ren reng rong rou ru ruan rui run ruo sa sai san sang se sen sha shan shang shao she shen sheng shi shou shu shuai shuang shui shun shuo si song su suan sui sun suo ta tai tan tang tao te teng ti tian tiao tie ting tong tou tu tuan tun tuo wa wan wang wei wen weng wo wu xi xia xian xiang xiao xie xin xing xiong xiu xu xuan xue xun ya yan yang yao ye yi yin ying yong you yu yuan yue yun zai zan ze zeng zha zhan zhang zhao zhe zhen zheng zhi zhong zhou zhu zhuang zhuo zi zong zou zu zuo
ae ah bae baek beom bok bom bong bum byeong byung chae cheol chul dae deok duk eun geon geun gi gil gwang gyeong gyu hae hee heon ho hoon hwa hwan hye hyeok hyeon hyo hyuk hyun il ja jae jeong jong joo joon kook kwan kwang kyeong kyo kyu kyun kyung mo moon mun myeong myung nam ok ra rae ri rim rin ro ryeong seo seok seon seong seul seung sik sil soo sook soon suk sung tae uk un won woo wook woon yeo yeol yeon yeong yoo yoon young yul
anh bich binh cam chau chien chinh cong cuc cuong dat diem diep dieu dinh doan duc dung duong duy gia giang hanh hau hien hiep hieu hoa hoai hoan hoang hue hung huong huu huy huyen khai khang khanh khiem khoa khoi khue kien kieu kim ky lam lien lieu linh loan loc loi luc luong ly manh minh my nga ngan nghi nghia ngoc nguyen nguyet nhan nhat nhi nhu nhung ninh oanh phat phi phong phu phuc phung phuong quang quoc quy quyen quynh sinh suong tam thach thai tham thang thanh thao thi thien thinh tho thoa thong thu thuan thuy thuong tien tin tinh toan tra tram trang tri trieu trinh trong truc trung truong tuan tue tung tuyen tuyet uyen van vi viet vinh vu vuong vy xuan yen`)

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

func eastGiven(w string) (ok, distinctive bool) {
	lw := textutil.Fold(w)
	if strings.Contains(lw, "-") {
		parts := strings.Split(lw, "-")
		if len(parts) != 2 || syllableCount(parts[0]) != 1 || syllableCount(parts[1]) != 1 {
			return false, false
		}
		return true, true
	}
	switch syllableCount(lw) {
	case 1:
		return true, false
	case 2:
		return true, len(lw) >= 5 && !syllablePairWords[lw]
	}
	return false, false
}

func familyFirst(text string, toks []nameTok, i int) (end, n int, canonical string) {
	t := toks[i]
	if t.shape != shapeTitle {
		return 0, 0, ""
	}
	fam := textutil.Fold(t.text)
	strong := familyFirstStrong[fam]
	if !strong && !familyFirstWeak[fam] {
		return 0, 0, ""
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
		return 0, 0, ""
	}
	end = toks[j-1].start + len(given[len(given)-1])
	if placeUse(text, t.start, end) {
		return 0, 0, ""
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
		accept = plain && lowerNameContext(text, t.start, end)
	}
	if !accept {
		return 0, 0, ""
	}
	return end, j - 1 - i, strings.Join(given, " ") + " " + t.text
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
