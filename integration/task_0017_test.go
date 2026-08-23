package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/httpapi"
)

func TestMissingQualityEvidenceMapsToConflict(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0017?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','open',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-17','green-methanol',100,'rejected','2030-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(rt, nil)
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"planner@example.test","password":"planner-pass"}`))
	loginResult := httptest.NewRecorder()
	handler.ServeHTTP(loginResult, login)
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginResult.Body.Bytes(), &session); err != nil || session.Token == "" {
		t.Fatalf("login status=%d body=%s error=%v", loginResult.Code, loginResult.Body.String(), err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/bunkering", strings.NewReader(`{"VesselID":"v","WindowID":"w","FuelLotID":"l","target_kg":25}`))
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Idempotency-Key", "task-17")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "fuel quality") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
