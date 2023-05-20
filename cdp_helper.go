package cdp_helper

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"log"
	"time"
)

type ContextWithCancel struct {
	Context context.Context
	Cancel  context.CancelFunc
}

type CdpHelper struct {
	// Allocator represents chromedp allocator with cancel func
	Allocator ContextWithCancel
	Browser   ContextWithCancel
	Current   *ContextWithCancel

	Timeout         time.Duration
	TextTimeout     time.Duration
	DownloadTimeout time.Duration
}

type Logger interface {
	Errorf(string, ...any)
	Debugf(string, ...any)
	Logf(string, ...any)
}

type DefaultLogger struct {
}

func (*DefaultLogger) Errorf(format string, args ...any) {
	log.Printf(format, args...)
}

func (*DefaultLogger) Debugf(format string, args ...any) {
	log.Printf(format, args...)
}

func (*DefaultLogger) Logf(format string, args ...any) {
	log.Printf(format, args...)
}

func NewBrowser(headless bool) *CdpHelper {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("headless", headless))

	allocator, allocatorCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserContext, browserCancel := chromedp.NewContext(allocator)

	helper := CdpHelper{
		Allocator: ContextWithCancel{
			Context: allocator,
			Cancel:  allocatorCancel,
		},
		Browser: ContextWithCancel{
			Context: browserContext,
			Cancel:  browserCancel,
		},
	}

	helper.Current = &helper.Browser
	helper.setDefault()

	return &helper
}

func NewRemoteBrowser(url string) *CdpHelper {
	remoteAllocator, remoteAllocatorCancel := chromedp.NewRemoteAllocator(context.Background(), url)
	remoteBrowserContext, remoteBrowserCancel := chromedp.NewContext(remoteAllocator)

	helper := CdpHelper{
		Allocator: ContextWithCancel{
			Context: remoteAllocator,
			Cancel:  remoteAllocatorCancel,
		},
		Browser: ContextWithCancel{
			Context: remoteBrowserContext,
			Cancel:  remoteBrowserCancel,
		},
	}

	helper.Current = &helper.Browser
	helper.setDefault()

	return &helper
}

func (h *CdpHelper) setDefault() {
	h.Timeout = 3 * time.Second
	h.TextTimeout = 1 * time.Second
	h.DownloadTimeout = 10 * time.Second
}

func (h *CdpHelper) WithTimeout(timeout time.Duration) {
	h.Timeout = timeout
}

func (h *CdpHelper) WithTextTimeout(timeout time.Duration) {
	h.TextTimeout = timeout
}

// NewBlankTab returns a new CdpHelper instance, and CdpHelper.Current points to the new tab
func (h *CdpHelper) NewBlankTab(targetId string) (*CdpHelper, error) {
	if targetId == "" {
		targetId = "_blank"
	}

	ch := chromedp.WaitNewTarget(h.Current.Context, func(info *target.Info) bool {
		return true
	})

	js := fmt.Sprintf(`window.open("about:blank", "%s")`, targetId)

	var res *runtime.RemoteObject
	defer func() {
		if res != nil {
			runtime.ReleaseObject(res.ObjectID)
		}
	}()

	err := chromedp.Run(h.Current.Context, chromedp.Evaluate(js, &res))
	if err != nil {
		return nil, err
	}

	id := <-ch
	targetContext, targetCancel := chromedp.NewContext(h.Current.Context, chromedp.WithTargetID(id))

	helper := CdpHelper{
		Allocator: h.Allocator,
		Browser:   h.Browser,
		Current: &ContextWithCancel{
			Context: targetContext,
			Cancel:  targetCancel,
		},
	}

	return &helper, nil
}

func (h *CdpHelper) Navigate(url string) error {
	return h.Run(chromedp.Navigate(url))
}

func (h *CdpHelper) NodeTextContent(sel any, opts ...chromedp.QueryOption) (string, error) {
	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, h.TextTimeout)
	defer timeoutCancel()
	var text string
	err := chromedp.Run(timeoutCtx, chromedp.TextContent(sel, &text, opts...))
	if err != nil {
		return "", err
	}

	return text, nil
}

func (h *CdpHelper) Nodes(sel any, opts ...chromedp.QueryOption) ([]*cdp.Node, error) {
	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, h.Timeout)
	defer timeoutCancel()

	var nodes []*cdp.Node
	err := chromedp.Run(timeoutCtx, chromedp.Nodes(sel, &nodes, opts...))

	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		err = chromedp.Run(timeoutCtx, dom.RequestChildNodes(node.NodeID).WithDepth(-1).WithPierce(true))
		if err != nil {
			return nil, err
		}
	}

	return nodes, err
}

func (h *CdpHelper) ChildNodes(parent *cdp.Node, cssSel string) ([]cdp.NodeID, error) {
	executor := h.NewTargetExecutor(h.Current.Context)
	nodeIDs, err := dom.QuerySelectorAll(parent.NodeID, cssSel).Do(executor)
	if err != nil {
		return nil, err
	}
	return nodeIDs, nil
}

func (h *CdpHelper) ChildNodeTextContent(parent *cdp.Node, cssSel string) (string, error) {
	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, h.TextTimeout)
	defer timeoutCancel()

	executor, childNodeID, err := h.ChildNode(timeoutCtx, parent.NodeID, cssSel)
	if err != nil {
		return "", err
	}

	var text string
	err = chromedp.TextContent([]cdp.NodeID{childNodeID}, &text, chromedp.ByNodeID).Do(executor)
	if err != nil {
		return "", err
	}

	return text, nil
}

func (h *CdpHelper) Download(path string, isNewTarget bool) (*chan string, context.Context, error) {
	done := make(chan string, 1)

	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, h.DownloadTimeout)
	listenEvent := func(ev any) {
		if v, ok := ev.(*browser.EventDownloadProgress); ok {
			if v.State == browser.DownloadProgressStateCompleted {
				done <- v.GUID
				close(done)
				timeoutCancel()
			}
		}
	}

	var executor context.Context
	if isNewTarget {
		chromedp.ListenBrowser(timeoutCtx, func(ev interface{}) {
			listenEvent(ev)
		})
		executor = h.NewBrowserExecutor(h.Current.Context)
	} else {
		chromedp.ListenTarget(timeoutCtx, func(ev interface{}) {
			listenEvent(ev)
		})
		executor = h.NewTargetExecutor(h.Current.Context)
	}

	err := browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
		WithDownloadPath(path).
		WithEventsEnabled(true).
		Do(executor)
	if err != nil {
		return nil, nil, err
	}

	return &done, timeoutCtx, nil
}

func (h *CdpHelper) Click(sel any, opts ...chromedp.QueryOption) error {
	return h.Run(chromedp.Click(sel, opts...))
}

func (h *CdpHelper) ClickChild(parent *cdp.Node, cssSel string, opts ...chromedp.MouseOption) error {
	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, h.Timeout)
	defer timeoutCancel()
	executor, childNodeID, err := h.ChildNode(timeoutCtx, parent.NodeID, cssSel)
	if err != nil {
		return err
	}

	var childNode *cdp.Node
	childNode, err = dom.DescribeNode().WithNodeID(childNodeID).Do(executor)
	if err != nil {
		return nil
	}
	childNode.NodeID = childNodeID

	return h.Run(chromedp.MouseClickNode(childNode, opts...))
}

func (h *CdpHelper) SendKeys(sel any, v string, opts ...chromedp.QueryOption) error {
	return h.Run(chromedp.SendKeys(sel, v, opts...))
}

func (h *CdpHelper) WaitReady(sel any, opts ...chromedp.QueryOption) error {
	return h.Run(chromedp.WaitReady(sel, opts...))
}

func (h *CdpHelper) Sleep(d time.Duration) error {
	return h.Run(chromedp.Sleep(d))
}

func (h *CdpHelper) Attributes(sel any, opts ...chromedp.QueryOption) (map[string]string, error) {
	var attributes map[string]string
	err := h.Run(chromedp.Attributes(sel, &attributes, opts...))
	if err != nil {
		return nil, err
	}

	return attributes, nil
}

func (h *CdpHelper) AttributesAll(sel any, opts ...chromedp.QueryOption) ([]map[string]string, error) {
	var attributes []map[string]string
	err := h.Run(chromedp.AttributesAll(sel, &attributes, opts...))
	if err != nil {
		return nil, err
	}

	return attributes, nil
}

func (h *CdpHelper) SetAttributeValue(sel any, name string, value string, opts ...chromedp.QueryOption) error {
	return h.Run(chromedp.SetAttributeValue(sel, name, value, opts...))
}

func (h *CdpHelper) SetAttributes(sel any, attributes map[string]string, opts ...chromedp.QueryOption) error {
	return h.Run(chromedp.SetAttributes(sel, attributes, opts...))
}

func (h *CdpHelper) Run(actions ...chromedp.Action) error {
	return chromedp.Run(h.Current.Context, actions...)
}

func (h *CdpHelper) RunWithTimeout(t time.Duration, actions ...chromedp.Action) error {
	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, t)
	defer timeoutCancel()
	return chromedp.Run(timeoutCtx, actions...)
}

func (h *CdpHelper) Tasks(actions ...chromedp.Action) error {
	return h.Run(actions...)
}

func (h *CdpHelper) ChildNode(ctx context.Context, parent cdp.NodeID, cssSel string) (context.Context, cdp.NodeID, error) {
	executor := h.NewTargetExecutor(ctx)
	var nodeID cdp.NodeID
	var err error
	if cssSel != "" {
		nodeID, err = dom.QuerySelector(parent, cssSel).Do(executor)
		if err != nil {
			return nil, 0, err
		}
		if nodeID == cdp.EmptyNodeID {
			return nil, 0, errors.New("empty nodeID")
		}
	} else {
		nodeID = parent
	}

	return executor, nodeID, nil
}

func (h *CdpHelper) HasChildNode(parent *cdp.Node, cssSel string) (cdp.NodeID, bool) {
	timeoutCtx, timeoutCancel := context.WithTimeout(h.Current.Context, h.Timeout)
	defer timeoutCancel()
	_, nodeID, err := h.ChildNode(timeoutCtx, parent.NodeID, cssSel)
	if err != nil {
		return 0, false
	}
	return nodeID, true
}

func (h *CdpHelper) NewBrowserExecutor(ctx context.Context) context.Context {
	c := chromedp.FromContext(h.Current.Context)
	executor := cdp.WithExecutor(ctx, c.Browser)
	return executor
}

func (h *CdpHelper) NewTargetExecutor(ctx context.Context) context.Context {
	c := chromedp.FromContext(h.Current.Context)
	executor := cdp.WithExecutor(ctx, c.Target)
	return executor
}
