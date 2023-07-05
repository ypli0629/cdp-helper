package cdp_helper

import (
	"context"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestCdpHelper_NewBlankTab(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	nh, err := b.NewBlankTab("")
	assert.Nil(t, err)
	assert.NotNil(t, nh)
	err = nh.Run(chromedp.Navigate("https://www.google.com"))
	assert.Nil(t, err)
	err = nh.Run(chromedp.Sleep(3 * time.Second))
	assert.Nil(t, err)
}

func TestCdpHelper_Navigate(t *testing.T) {
	err := NewBrowser(true).Navigate("https://www.baidu.com")
	assert.Nil(t, err)
}

func TestCdpHelper_Nodes(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	nodes, err := b.Nodes(`//*[@id="hotsearch-content-wrapper"]/li`)
	assert.Nil(t, err)
	assert.Greater(t, len(nodes), 0)
	assert.Equal(t, len(nodes[0].Children), int(nodes[0].ChildNodeCount))
	assert.Equal(t, len(nodes[0].Children[0].Children), int(nodes[0].Children[0].ChildNodeCount))
}

func TestCdpHelper_NodeText(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	text, err := b.NodeTextContent(`//*[@id="s-top-left"]/a[1]`)
	assert.Nil(t, err)
	assert.NotEmpty(t, text)
}

func TestCdpHelper_ChildNodeText(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	nodes, err := b.Nodes(`//*[@id="hotsearch-content-wrapper"]/li`)
	assert.Nil(t, err)
	for _, node := range nodes {
		var text string
		text, err = b.ChildNodeTextContent(node, `a > span.title-content-title`)
		assert.Nil(t, err)
		assert.NotEmpty(t, text)
	}
}

func TestCdpHelper_Download(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://github.com/chromedp/examples")
	assert.Nil(t, err)
	err = b.Click(`//get-repo//summary`, chromedp.NodeReady)
	assert.Nil(t, err)
	ch, timeoutCtx, cancel, err := b.Download("/home/ypli/Downloads", false)
	defer cancel()
	assert.Nil(t, err)
	err = b.Click(`//get-repo//a[contains(@data-ga-click, "download zip")]`, chromedp.NodeVisible)
	assert.Nil(t, err)
	select {
	case <-timeoutCtx.Done():
		assert.Equal(t, context.Canceled, timeoutCtx.Err())
	case id := <-*ch:
		assert.NotEmpty(t, id)
		err = os.Remove(path.Join("/home/ypli/Downloads", id))
		assert.Nil(t, err)
	}
	assert.Nil(t, err)
}

func TestCdpHelper_DownloadBrowser(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://github.com/chromedp/examples")
	assert.Nil(t, err)
	err = b.Click(`//get-repo//summary`, chromedp.NodeReady)
	assert.Nil(t, err)
	ch, timeoutCtx, cancel, err := b.Download("/home/ypli/Downloads", true)
	defer cancel()
	assert.Nil(t, err)
	err = b.Click(`//get-repo//a[contains(@data-ga-click, "download zip")]`, chromedp.NodeVisible)
	assert.Nil(t, err)
	select {
	case <-timeoutCtx.Done():
		assert.Equal(t, context.Canceled, timeoutCtx.Err())
	case id := <-*ch:
		assert.NotEmpty(t, id)
		err = os.Remove(path.Join("/home/ypli/Downloads", id))
		assert.Nil(t, err)
	}
	assert.Nil(t, err)
}

func TestCdpHelper_HasChildNode(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate(`https://github.com/chromedp/examples`)
	assert.Nil(t, err)
	nodes, err := b.Nodes(`//*[@id="repository-container-header"]/div[1]/div[1]/div/strong`)
	assert.Nil(t, err)
	nodeID, has := b.HasChildNode(nodes[0], `a`)
	assert.True(t, has)
	assert.NotZero(t, nodeID)
	nodeID, has = b.HasChildNode(nodes[0], `a1`)
	assert.False(t, has)
	assert.Zero(t, nodeID)
}

func TestChildNode(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate(`https://element.eleme.cn/#/zh-CN/component/select`)
	assert.Nil(t, err)
	err = b.Click(`//*[@id="app"]/div[2]/div/div[1]/div/div/div[2]/section/div[1]/div[1]/div/div`)
	assert.Nil(t, err)
	err = b.Sleep(3 * time.Second)
	assert.Nil(t, err)
	nodes, err := b.Nodes(`div.el-select-dropdown`)
	assert.Nil(t, err)
	err = b.Sleep(3 * time.Second)
	assert.Nil(t, err)
	var exist bool
	for _, node := range nodes {
		style, err := b.ComputedStyle([]cdp.NodeID{node.NodeID}, chromedp.ByNodeID)
		assert.Nil(t, err)
		if v, ok := style["display"]; ok {
			t.Log(v)
			if v != "none" {
				exist = true
				childNodes, err := b.ChildNodes(node, `ul > li`)
				assert.Nil(t, err)
				for _, childNode := range childNodes {
					content, err := b.ChildNodeTextContent(&cdp.Node{NodeID: childNode}, `span`)
					assert.Nil(t, err)
					if content == "龙须面" {
						err := b.Click([]cdp.NodeID{childNode}, chromedp.ByNodeID)
						assert.Nil(t, err)
						b.Sleep(3 * time.Second)
						b.ScreenShot("", "龙须面.png")
					}
				}
			}
		}
	}
	assert.True(t, exist)
}

func TestCdpHelper_ScreenShot(t *testing.T) {
	b := NewBrowser(true)
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	err = b.ScreenShot("", "baidu.png")
	assert.Nil(t, err)
}

func TestCdpHelper_Upload(t *testing.T) {
	//b := NewRemoteBrowser(RemoteBrowserOption{URL: "ws://127.0.0.1:9222", Logger: &DefaultLogger{}})
	b := NewBrowser(true)
	_ = b.Navigate("")
	_ = b.NavigateWithTimeout("https://element.eleme.cn/#/zh-CN/component/upload", 3*time.Second)
	err := b.WaitReady(`//*[@id="app"]/div[2]/div/div[1]/div/div/div[2]/section/div[1]/div[1]/div/div/div[1]/input`)
	assert.Nil(t, err)
	fp, err := filepath.Abs("/downloads/笔记.txt")
	//fp, err := filepath.Abs("/home/ypli/Downloads/test.txt")
	assert.Nil(t, err)
	_ = b.ScreenShot("./images", "upload.png")
	err = b.Upload(`//*[@id="app"]/div[2]/div/div[1]/div/div/div[2]/section/div[1]/div[1]/div/div/div[1]/input`, []string{fp})
	assert.Nil(t, err)
	err = b.Sleep(6 * time.Second)
	_ = b.ScreenShot("./images", "uploaded.png")
	assert.Nil(t, err)
}

func TestCdpHelper_ListenRequest(t *testing.T) {
	b := NewBrowser(true)
	ch := b.ListenRequest("/sugrec")
	b.Navigate("https://www.baidu.com")
	select {
	case <-time.NewTimer(5 * time.Second).C:
		assert.Fail(t, "timeout")
	case data := <-ch:
		print(string(data))
		assert.Greater(t, len(data), 0)
	}
}
