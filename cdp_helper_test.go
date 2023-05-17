package cdp_helper

import (
	"context"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
	"time"
)

func TestCdpHelper_NewBlankTab(t *testing.T) {
	b := NewBrowser()
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
	err := NewBrowser().Navigate("https://www.baidu.com")
	assert.Nil(t, err)
}

func TestCdpHelper_Nodes(t *testing.T) {
	b := NewBrowser()
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	nodes, err := b.Nodes(`//*[@id="hotsearch-content-wrapper"]/li`)
	assert.Nil(t, err)
	assert.Greater(t, len(nodes), 0)
	assert.Equal(t, len(nodes[0].Children), int(nodes[0].ChildNodeCount))
}

func TestCdpHelper_NodeText(t *testing.T) {
	b := NewBrowser()
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	text, err := b.NodeText(`//*[@id="s-top-left"]/a[1]`)
	assert.Nil(t, err)
	assert.NotEmpty(t, text)
}

func TestCdpHelper_ChildNodeText(t *testing.T) {
	b := NewBrowser()
	err := b.Navigate("https://www.baidu.com")
	assert.Nil(t, err)
	nodes, err := b.Nodes(`//*[@id="hotsearch-content-wrapper"]/li`)
	assert.Nil(t, err)
	for _, node := range nodes {
		var text string
		text, err = b.ChildNodeText(node, `a > span.title-content-title`)
		assert.Nil(t, err)
		assert.NotEmpty(t, text)
	}
}

func TestCdpHelper_Download(t *testing.T) {
	b := NewBrowser()
	err := b.Navigate("https://github.com/chromedp/examples")
	assert.Nil(t, err)
	err = b.Click(`//get-repo//summary`, chromedp.NodeReady)
	assert.Nil(t, err)
	ch, timeoutCtx, err := b.Download("/home/ypli/Downloads", false)
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
	b := NewBrowser()
	err := b.Navigate("https://github.com/chromedp/examples")
	assert.Nil(t, err)
	err = b.Click(`//get-repo//summary`, chromedp.NodeReady)
	assert.Nil(t, err)
	ch, timeoutCtx, err := b.Download("/home/ypli/Downloads", true)
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
