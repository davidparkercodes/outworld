package surrogate

import "strings"

type Origin int

const (
	OriginNone Origin = iota
	OriginVietnamese
	OriginChinese
	OriginKorean
)

var vietnameseFamilies = strings.Fields(`Huynh Trinh Luong Doan Quach Nghiem Khuat Phung Vuong Khuong Thach Tieu Khuc Khuu`)

var vietnameseGiven = strings.Fields(`
Cuong Hieu Linh Ngoc Nhung Thao Tuan Tuyet Xuan Quynh Thuy Huyen Hien Oanh Diep Toan Nghia Thinh
Duyen Nguyet Thuong Hoai Uyen Kiet Nhan Khang Thien Triet Khoi Quyen Suong
`)

var chineseFamilies = strings.Fields(`Zeng Shen Zheng Xiong Duan Meng Qian Cheng Shao Geng Kuang Qiao Weng Zang Luan Bian Zhuo Ruan Jiao Zhai Ouyang Zhuge Xiahou Murong`)

var chineseSyllables = strings.Fields(`Ning Chun Zhen Zhong Shuang Guang Huan Peng Ying Jian Rong Xing Qiong Shuo Jiong Ling Leng`)

var chineseGiven = strings.Fields(`
Weiming Xiaoling Jianhua Yuting Zhiwei Meiling Guoqiang Xiuying Haoran Yuxuan Zihan Jiahui Minghao Shufen Lihua Qiuyue Wenjie
Yifan Ruoxi Chenxi Junjie Hongmei Zhihao Peiyu Yanting Tianyu Jingyi Kaixin Xiaoyan Guohua Lixin Yaling
`)

var koreanFamilies = strings.Fields(`Yoon Jang Hwang Jeon Baek Seong Byun Chae Cheon Maeng Seol Yeom Myeong Gyeon Jegal Namgung Hwangbo Seonu Sagong Dokgo Pyeon Geum`)

var koreanSyllables = strings.Fields(`Hoon Yeon Hyun Kyung Seok Joon Seung Hwan Cheol Jeong Myung Hyeon Yeong Chul Geun Hyuk Sook Byung Kyun Woong Ryeol Beom Gyeong Hyeok`)

var vietnameseFamilyKeys = wordKeys(`nguyen tran pham hoang huynh phan dang bui ngo duong le do ho vo vu ly trinh truong dinh luong doan thai kieu`)

var koreanFamilyKeys = wordKeys(`kim lee park choi jung kang cho yoon jang lim han shin seo kwon hwang jeon baek shim seong byun chae noh oh ahn song yoo hong`)

var fictionalEast = map[string]bool{}

func init() {
	for _, list := range [][]string{vietnameseFamilies, vietnameseGiven, chineseFamilies, chineseSyllables, chineseGiven, koreanFamilies, koreanSyllables} {
		for _, n := range list {
			fictionalEast[strings.ToLower(n)] = true
		}
	}
}

func wordKeys(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

func IsEastMarker(word string) bool {
	lw := strings.ToLower(word)
	return lw == "van" || lw == "thi"
}

func IsBundledEastName(s string) bool { return fictionalEast[strings.ToLower(s)] }

func EastOrigin(family string, given []string) Origin {
	fam := strings.ToLower(family)
	if len(given) >= 2 && IsEastMarker(given[0]) {
		return OriginVietnamese
	}
	hyphen := false
	for _, g := range given {
		hyphen = hyphen || strings.Contains(g, "-")
	}
	switch {
	case vietnameseFamilyKeys[fam]:
		return OriginVietnamese
	case koreanFamilyKeys[fam]:
		return OriginKorean
	case hyphen && fam != "li" && fam != "wang" && fam != "zhang" && fam != "chen" && fam != "liu":
		return OriginKorean
	}
	return OriginChinese
}

func (g *Generator) EastFamilyName(origin Origin, real string, attempt int) string {
	list := chineseFamilies
	switch origin {
	case OriginVietnamese:
		list = vietnameseFamilies
	case OriginKorean:
		list = koreanFamilies
	}
	if attempt >= 3*len(list) {
		return g.FamilyName(real, attempt)
	}
	return g.stream("east-family", real, attempt).pick(list)
}

func (g *Generator) EastGivenName(origin Origin, real string, syllables, attempt int) string {
	r := g.stream("east-given", real, attempt)
	switch origin {
	case OriginVietnamese:
		if attempt < 3*len(vietnameseGiven) {
			return r.pick(vietnameseGiven)
		}
	case OriginKorean:
		if attempt < 3*len(koreanSyllables) {
			first := r.pick(koreanSyllables)
			if syllables < 2 {
				return first
			}
			return first + strings.ToLower(r.pick(koreanSyllables))
		}
	default:
		if syllables < 2 && attempt < 3*len(chineseSyllables) {
			return r.pick(chineseSyllables)
		}
		if syllables >= 2 && attempt < 3*len(chineseGiven) {
			return r.pick(chineseGiven)
		}
	}
	return g.GivenNameFor(real, "", attempt)
}

func (g *Generator) Initial(seed, not string) string {
	r := g.stream("initial", seed, 0)
	letters := "abcdefghijklmnopqrstuvwxyz"
	not = strings.ToLower(not)
	for {
		c := letters[r.intn(len(letters))]
		if not == "" || c != not[0] {
			return string(c)
		}
	}
}
