package browser

import (
	"context"
	"encoding/json"
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

	for _, a := range actions {
		switch a.Action {
		case "click":
			tasks = append(tasks, chromedp.Click(a.Selector, chromedp.NodeVisible))
		case "wait":
			tasks = append(tasks, chromedp.WaitVisible(a.Selector))
		case "type":
			tasks = append(tasks, chromedp.SendKeys(a.Selector, a.Value))
		}
	}

	if err := chromedp.Run(timeoutCtx, tasks); err != nil {
		return nil, errors.Wrap(err, "failed to execute browser actions")
	}

	duration := time.Since(start)

	res := &http.Result{
		Response: http.Response{
			StatusCode: 200,
			Body:       []byte("OK"),
		},
		Timing: http.Timing{
			Phases: http.TimingPhases{
				Total: duration,
			},
		},
	}

	return res, nil
}
