package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type SkysmartService struct {
	httpClient *http.Client
	baseURL    string
}

func NewSkysmartService() *SkysmartService {
	return &SkysmartService{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api-edu.skysmart.ru/api/v1",
	}
}

type SkysmartAnswer struct {
	TaskNumber   int      `json:"task_number"`
	Question     string   `json:"question"`
	FullQuestion string   `json:"full_question"`
	Answers      []string `json:"answers"`
}

func (s *SkysmartService) GetAnswers(ctx context.Context, taskHash string) ([]SkysmartAnswer, error) {
	// Get room info
	roomURL := fmt.Sprintf("%s/task/preview-by-task-hash/%s", s.baseURL, taskHash)
	req, err := http.NewRequestWithContext(ctx, "GET", roomURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch room: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var roomData struct {
		Meta struct {
			StepUUIDs []string `json:"stepUuids"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &roomData); err != nil {
		return nil, fmt.Errorf("parse room data: %w", err)
	}

	answers := make([]SkysmartAnswer, 0, len(roomData.Meta.StepUUIDs))
	for i, stepUUID := range roomData.Meta.StepUUIDs {
		answer, err := s.getStepAnswer(ctx, stepUUID, i+1)
		if err != nil {
			continue
		}
		answers = append(answers, *answer)
	}

	return answers, nil
}

func (s *SkysmartService) getStepAnswer(ctx context.Context, stepUUID string, taskNum int) (*SkysmartAnswer, error) {
	stepURL := fmt.Sprintf("%s/content/step/load", s.baseURL)

	payload := fmt.Sprintf(`{"stepUuid":"%s"}`, stepUUID)
	req, err := http.NewRequestWithContext(ctx, "POST", stepURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stepData struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &stepData); err != nil {
		return nil, err
	}

	// Parse the HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(stepData.Content))
	if err != nil {
		return nil, err
	}

	answer := &SkysmartAnswer{
		TaskNumber: taskNum,
	}

	// Extract question
	doc.Find(".task-question, .question").Each(func(_ int, sel *goquery.Selection) {
		answer.Question = strings.TrimSpace(sel.Text())
	})

	// Extract answers from various question types
	doc.Find("[data-answer]").Each(func(_ int, sel *goquery.Selection) {
		dataAnswer, exists := sel.Attr("data-answer")
		if !exists {
			return
		}
		// Try base64 decode
		decoded, err := base64.StdEncoding.DecodeString(dataAnswer)
		if err == nil {
			answer.Answers = append(answer.Answers, string(decoded))
		} else {
			answer.Answers = append(answer.Answers, dataAnswer)
		}
	})

	answer.FullQuestion = doc.Text()

	return answer, nil
}
