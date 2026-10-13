package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/pkg/errors"
)

type Service interface {
	Probe(ctx context.Context, url string, scriptJSON string, timeout time.Duration) (*http.Result, error)
}

type ServiceImpl struct{}

func NewService() Service {
	return &ServiceImpl{}
}

type Action struct {
	Action   string `json:"action"`
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

func (s *ServiceImpl) Probe(ctx context.Context, url string, scriptJSON string, timeout time.Duration) (*http.Result, error) {
	// Parse the actions from JSON script
	var actions []Action
	if scriptJSON != "" {
		if err := json.Unmarshal([]byte(scriptJSON), &actions); err != nil {
			return nil, errors.Wrap(err, "failed to parse browser script")
		}
	}

	start := time.Now()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	timeoutCtx, cancelTimeout := context.WithTimeout(taskCtx, timeout)
	defer cancelTimeout()

	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
	}

	for idx, a := range actions {
		actName := strings.ToLower(a.Action)
		switch actName {
		case "click":
			tasks = append(tasks, chromedp.Click(a.Selector, chromedp.NodeVisible))
		case "wait", "wait_visible":
			tasks = append(tasks, chromedp.WaitVisible(a.Selector))
		case "type", "fill":
			tasks = append(tasks, chromedp.SendKeys(a.Selector, a.Value))
		case "navigate":
			tasks = append(tasks, chromedp.Navigate(a.Value))
		case "assert_text", "text_contains":
			var extractedText string
			selector := a.Selector
			expected := a.Value
			stepNum := idx + 1
			tasks = append(tasks,
				chromedp.WaitVisible(selector),
				chromedp.Text(selector, &extractedText),
				chromedp.ActionFunc(func(c context.Context) error {
					if !strings.Contains(extractedText, expected) {
						return fmt.Errorf("step %d assert_text failed: element '%s' content '%s' does not contain '%s'", stepNum, selector, extractedText, expected)
					}
					return nil
				}),
			)
		case "sleep", "delay":
			ms, _ := strconv.Atoi(a.Value)
			if ms <= 0 {
				ms = 500
			}
			tasks = append(tasks, chromedp.Sleep(time.Duration(ms)*time.Millisecond))
		case "press_key", "key":
			tasks = append(tasks, chromedp.KeyEvent(a.Value))
		case "scroll", "hover":
			tasks = append(tasks, chromedp.ScrollIntoView(a.Selector))
		case "evaluate", "eval":
			tasks = append(tasks, chromedp.Evaluate(a.Value, nil))
		case "screenshot":
			var buf []byte
			tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
		}
	}

	if err := chromedp.Run(timeoutCtx, tasks); err != nil {
		return nil, errors.Wrap(err, "failed to execute synthetic browser actions")
	}

	duration := time.Since(start)

	res := &http.Result{
		Response: http.Response{
			StatusCode: 200,
			Body:       []byte("OK: Synthetic browser journey completed successfully"),
		},
		Timing: http.Timing{
			Phases: http.TimingPhases{
				Total: duration,
			},
		},
	}

	return res, nil
}
