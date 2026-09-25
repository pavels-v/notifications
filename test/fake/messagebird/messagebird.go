package messagebird

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	Originator string   `json:"originator"`
	Recipients []string `json:"recipients"`
	Body       string   `json:"body"`
}

type failure struct {
	status      int
	code        int
	description string
}

type Fake struct {
	accessKey string
	sent      atomic.Int64

	mu       sync.Mutex
	requests []Request
	fail     *failure
	drop     bool
}

func New(accessKey string) *Fake {
	return &Fake{accessKey: accessKey}
}

func (f *Fake) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /messages", f.send)
	return mux
}

func (f *Fake) FailWith(status, code int, description string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.fail = &failure{status: status, code: code, description: description}
}

func (f *Fake) DropConnections() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.drop = true
}

func (f *Fake) Requests() []Request {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]Request, len(f.requests))
	copy(out, f.requests)
	return out
}

func (f *Fake) send(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "AccessKey "+f.accessKey {
		writeErrors(w, http.StatusUnauthorized, 2, "Request not allowed (incorrect access_key)")
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrors(w, http.StatusUnprocessableEntity, 9, "no (correct) recipients found")
		return
	}

	f.mu.Lock()
	f.requests = append(f.requests, req)
	fail, drop := f.fail, f.drop
	f.mu.Unlock()

	if drop {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			writeErrors(w, http.StatusInternalServerError, 0, "this server cannot drop a connection")
			return
		}

		conn, _, err := hijacker.Hijack()
		if err != nil {
			return
		}
		_ = conn.Close()
		return
	}

	if fail != nil {
		writeErrors(w, fail.status, fail.code, fail.description)
		return
	}
	if len(req.Recipients) == 0 {
		writeErrors(w, http.StatusUnprocessableEntity, 9, "no (correct) recipients found")
		return
	}

	writeAccepted(w, req, f.sent.Add(1))
}

type item struct {
	Recipient      int64  `json:"recipient"`
	Status         string `json:"status"`
	StatusDatetime string `json:"statusDatetime"`
}

type accepted struct {
	ID                string         `json:"id"`
	HRef              string         `json:"href"`
	Direction         string         `json:"direction"`
	Type              string         `json:"type"`
	Originator        string         `json:"originator"`
	Body              string         `json:"body"`
	Reference         *string        `json:"reference"`
	Validity          *int           `json:"validity"`
	Gateway           int            `json:"gateway"`
	TypeDetails       map[string]any `json:"typeDetails"`
	Datacoding        string         `json:"datacoding"`
	MClass            int            `json:"mclass"`
	ScheduledDatetime *string        `json:"scheduledDatetime"`
	CreatedDatetime   string         `json:"createdDatetime"`
	Recipients        struct {
		TotalCount               int    `json:"totalCount"`
		TotalSentCount           int    `json:"totalSentCount"`
		TotalDeliveredCount      int    `json:"totalDeliveredCount"`
		TotalDeliveryFailedCount int    `json:"totalDeliveryFailedCount"`
		Items                    []item `json:"items"`
	} `json:"recipients"`
}

const datetimeLayout = "2006-01-02T15:04:05-07:00"

func writeAccepted(w http.ResponseWriter, req Request, n int64) {
	now := time.Now().UTC().Format(datetimeLayout)
	id := fmt.Sprintf("fake-%d", n)

	var resp accepted
	resp.ID = id
	resp.HRef = "https://rest.messagebird.com/messages/" + id
	resp.Direction = "mt"
	resp.Type = "sms"
	resp.Originator = req.Originator
	resp.Body = req.Body
	resp.Reference = nil
	resp.Validity = nil
	resp.ScheduledDatetime = nil
	resp.Gateway = 10
	resp.TypeDetails = map[string]any{}
	resp.Datacoding = "plain"
	resp.MClass = 1
	resp.CreatedDatetime = now

	resp.Recipients.TotalCount = len(req.Recipients)
	resp.Recipients.TotalSentCount = len(req.Recipients)
	resp.Recipients.TotalDeliveredCount = 0
	resp.Recipients.TotalDeliveryFailedCount = 0

	for _, to := range req.Recipients {
		resp.Recipients.Items = append(resp.Recipients.Items, item{
			Recipient:      msisdn(to),
			Status:         "sent",
			StatusDatetime: now,
		})
	}

	write(w, http.StatusCreated, resp)
}

func writeErrors(w http.ResponseWriter, status, code int, description string) {
	write(w, status, map[string]any{
		"errors": []map[string]any{{"code": code, "description": description}},
	})
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func msisdn(to string) int64 {
	n, _ := strconv.ParseInt(strings.TrimPrefix(to, "+"), 10, 64)
	return n
}
