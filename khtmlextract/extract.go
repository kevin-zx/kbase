// Package khtmlextract 主要作用是导出网页正文，一般适用于新闻页。
package khtmlextract

import (
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

// NodeInfo 存储每个结点的各种信息
type NodeInfo struct {
	T  int // 结点字符串字数 node text len
	LT int // 结点带连接的字符串的字数
	// 结点标签数
	TG    int
	LTG   int     // 结点带连接的标签数
	PNum  int     // p 标签的结点数
	TD    float64 // 文本密度
	Score float64 // 最终得分
	Sb    int     // 符号长度
	SbD   float64 // 符号密度
}

// Article 存储提取出来的文章信息
type Article struct {
	Title       string
	Summary     string // summary
	ContentText string
	ContentHTML string
	Score       float64
	TextLength  int
}

// ExtractArticle 从 html 中提取文章信息
func ExtractArticle(html string) (*Article, error) {
	infoMap, err := calculate(html)
	if err != nil {
		return nil, err
	}
	maxScore := 0.0
	avgScore := 0.0
	sumScore := 0.0
	var maxNode *goquery.Selection
	//if len(infoMap) <= 5 {
	//	return nil, fmt.Errorf("can't find page content")
	//}
	//var maxInfo *NodeInfo
	for node, info := range infoMap {
		//h, _ := node.Html()
		//fmt.Println("len:", len(strings.Split(node.Text(), "")), info.Score, node.Text(), h)
		//fmt.Println("--------------------------------------------------------------------")
		if info.Score > maxScore {
			maxScore = info.Score
			maxNode = node
			//maxInfo = info

		}
		sumScore += info.Score
	}
	if len(infoMap) == 0 {
		return nil, fmt.Errorf("can't find page content")
	}
	avgScore = sumScore / float64(len(infoMap))
	if maxNode == nil {
		return nil, fmt.Errorf("can't find page content")
	}
	maxNode = removeSuccessiveLink(maxNode)
	if maxNode == nil {
		return nil, fmt.Errorf("extract article err, can't get content node")
	}
	a := Article{}
	a.ContentHTML, err = goquery.OuterHtml(maxNode)
	if err != nil {
		return nil, err
	}
	a.ContentText = strings.ReplaceAll(getClearTxt(maxNode), "\n\n", "\n")
	a.Title, a.Summary, err = getArticleInfo(html)
	if err != nil {
		return nil, err
	}
	a.Score = maxScore / avgScore
	a.TextLength = utf8.RuneCountInString(a.ContentText)
	return &a, err
}

// 有些网站中会把作者，来源等加上链接放入到文章中，还有就是一个div下面的第一级子节点就分布着标题，内容，上一篇下一篇，推荐等。
// 这些链接主要特点是有连续性，然后，goquery dom子节点的遍历也是自上而下的，基于以上
// 所以这个函数做清除用
func removeSuccessiveLink(node *goquery.Selection) *goquery.Selection {
	var successionAs []*goquery.Selection
	var needRemoveElements []*goquery.Selection
	lastIndex := 0

	node.Children().Each(func(i int, subNode *goquery.Selection) {
		subNodeTxt := getClearTxt(subNode)
		// 有些特殊特征的元素可以被删除
		if len(strings.Split(subNodeTxt, "")) < 64 && (strings.HasPrefix(strings.TrimSpace(subNodeTxt), "上一篇") || strings.HasPrefix(strings.TrimSpace(subNodeTxt), "下一篇")) {
			needRemoveElements = append(needRemoveElements, subNode)
			lastIndex = i
			return
		}
		// 当前元素是链接元素
		if subNode.Is("a") {
			lastIndex = i
			successionAs = append(successionAs, subNode)
			return
		}
		aChildren := subNode.Find("a")
		if aChildren.Length() > 0 {
			tr := float64(len(getClearTxt(aChildren))) / float64(len(subNodeTxt))
			if tr >= 0.6 || aChildren.Length() >= 3 {
				// 当前元素含有一个 a 元素
				if aChildren.Length() == 1 {
					lastIndex = i
					successionAs = append(successionAs, aChildren)
				}
				// 当前元素含有多个 a 元素
				if aChildren.Length() > 1 {
					needRemoveElements = append(needRemoveElements, subNode)
					lastIndex = i
					return
				}
			}
		}

		//这里不连续了
		if lastIndex != 0 && lastIndex != i {
			// 存储了 2个以上的链接 符合删除的标准
			if len(successionAs) > 1 {
				needRemoveElements = append(needRemoveElements, successionAs...)
			}
			// clear
			successionAs = []*goquery.Selection{}
			lastIndex = 0
		}

	})
	for _, element := range needRemoveElements {
		element.Remove()
	}
	return node
}

// getArticleInfo 解析 html 返回 title summary 信息
func getArticleInfo(html string) (title string, summary string, err error) {
	//
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "", err
	}
	title = getClearTxt(doc.Find("H1"))
	if title == "" {
		title = doc.Find("title").Text()
	}
	doc.Find("meta").EachWithBreak(func(_ int, meta *goquery.Selection) bool {
		if strings.Contains(meta.AttrOr("name", ""), "description") || strings.Contains(meta.AttrOr("property", ""), "description") {
			summary = meta.AttrOr("content", "")
			return false
		}
		return true
	})
	return
}

// compute node info
func computeInfo(node *goquery.Selection, nodeMap map[*goquery.Selection]*NodeInfo) *NodeInfo {
	var nodeInfo = &NodeInfo{}
	if node.Children().Length() > 0 {
		if node.Is("p") {
			nodeInfo.PNum++
		} else {

			// 有子节点的情况下
			node.Children().Each(func(_ int, child *goquery.Selection) {
				// 计算当前结点的子节点的各种基础信息
				// calculate all child node base info
				childNodeInfo := computeInfo(child, nodeMap)
				//nodeInfo.T += childNodeInfo.T
				nodeInfo.LT += childNodeInfo.LT
				nodeInfo.TG += childNodeInfo.TG
				nodeInfo.LTG += childNodeInfo.LTG
				nodeInfo.PNum += childNodeInfo.PNum
				//nodeInfo.Sb += childNodeInfo.Sb
			})

		}
		nodeInfo.T = len(getClearTxt(node))
		if nodeInfo.T > 0 {
			for _, r := range getClearTxt(node) {
				if unicode.IsPunct(r) {
					nodeInfo.Sb++
				}
			}
		}
		nodeInfo.TG++
		nodeMap[node] = nodeInfo
	} else {
		// 没有子节点的情况下
		// 累加指定结点 (a tag, p tag) 的信息
		// accumulate special tags params
		if node.Is("a") {
			nodeInfo.LTG++
			nodeInfo.LT = len(getClearTxt(node))
		} else if node.Is("p") {
			nodeInfo.PNum++
		}
		nodeInfo.TG = 1

	}

	return nodeInfo
}

func calculate(html string) (map[*goquery.Selection]*NodeInfo, error) {
	nodeMap := make(map[*goquery.Selection]*NodeInfo)
	doc, err := removeScriptAndStyle(html)
	if err != nil {
		return nil, err
	}
	body := doc.Find("body")
	computeInfo(body, nodeMap)
	sum := 0.0

	for _, info := range nodeMap {
		// 计算文本密度
		// calculate text density
		td := (float64(info.T-info.LT) / float64(info.TG-info.LTG+1)) * math.Log10(float64(info.TG-info.LTG+1))
		info.TD = td
		sum += td
		info.SbD = float64(info.T-info.LT) / float64(info.Sb+1)
	}
	nodeInfoCount := float64(len(nodeMap))
	avg := sum / nodeInfoCount

	// 计算文本密度标准差
	// calculate text density's standard deviation
	sdp := 0.0
	for _, info := range nodeMap {
		sdp += math.Pow(info.TD-avg, 2) / (nodeInfoCount)

	}
	sd := math.Sqrt(sdp)

	sdLog := math.Log(sd)
	if sd == 0 {
		sdLog = 0
	}
	for _, info := range nodeMap {
		// 计算结点信息
		// calculate node info score
		// latex formula: score = \log_{}{SD}*ND_i*\log_{10}{(PNum_i)}*\log_{}{SbD_i}
		info.Score = sdLog * info.TD * math.Log(float64(info.PNum+1)) * math.Log(info.SbD+1)
	}

	return nodeMap, nil
}

func removeScriptAndStyle(html string) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		s.Remove()
	})

	doc.Find("style").Each(func(_ int, s *goquery.Selection) {
		s.Remove()
	})
	return doc, nil
}
func getClearTxt(selection *goquery.Selection) string {
	content := selection.Text()
	for strings.Contains(content, " \n") {
		content = strings.ReplaceAll(content, " \n", "\n")
	}
	for strings.Contains(content, "\n\t") {
		content = strings.ReplaceAll(content, "\n\t", "\n")
	}
	for strings.Contains(content, "\n ") {
		content = strings.ReplaceAll(content, "\n ", "\n")
	}
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}
	for strings.Contains(content, "\t\t") {
		content = strings.ReplaceAll(content, "\t\t", "\t")
	}
	for strings.Contains(content, "\n\n") {
		content = strings.ReplaceAll(content, "\n\n", "\n")
	}
	for strings.Contains(content, "\t\n") {
		content = strings.ReplaceAll(content, "\t\n", "\n")
	}
	return content
}
