package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/degdb/degdb/protocol"
)

func init() {
	dir, err := ioutil.TempDir("", "degdb-test-files")
	if err != nil {
		log.Fatal(err)
	}
	KeyFilePath = dir + "/degdb-%d.key"
	DatabaseFilePath = dir + "/degdb-%d.db"
}

func testServer(t *testing.T) *server {
	s, err := newServer(0, nil, diskAllocated)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestHTTP(t *testing.T) {
	t.Parallel()

	s := testServer(t)
	go s.network.Listen()
	time.Sleep(10 * time.Millisecond)
	base := fmt.Sprintf("http://localhost:%d", s.network.Port)

	localPeerJSON, err := json.Marshal(s.network.LocalPeer())
	if err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		path string
		want string
	}{
		{
			"/api/v1/info",
			string(localPeerJSON),
		},
		{
			"/api/v1/myip",
			"::1",
		},
		{
			"/api/v1/triples",
			"[]",
		},
		{
			"/api/v1/peers",
			"[]",
		},
	}

	for i, td := range testData {
		url := base + td.path
		resp, err := http.Get(url)
		if err != nil {
			t.Error(err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		bodyTrim := strings.TrimSpace(string(body))
		wantTrim := strings.TrimSpace(td.want)
		if bodyTrim != wantTrim {
			t.Errorf("%d. http.Get(%+v) = %+v; not %+v", i, td.path, bodyTrim, wantTrim)
		}
	}
}

var testTriples = []*protocol.Triple{
	{
		Subj: "/m/02mjmr",
		Pred: "/type/object/name",
		Obj:  "Barack Obama",
	},
	{
		Subj: "/m/02mjmr",
		Pred: "/type/object/type",
		Obj:  "/people/person",
	},
	{
		Subj: "/m/0hume",
		Pred: "/type/object/name",
		Obj:  "Hume",
	},
	{
		Subj: "/m/0hume",
		Pred: "/type/object/type",
		Obj:  "/organization/team",
	},
}

func TestInsertAndRetreiveTriples(t *testing.T) {
	t.Parallel()

	s := testServer(t)
	go s.network.Listen()

	time.Sleep(10 * time.Millisecond)
	base := fmt.Sprintf("http://localhost:%d", s.network.Port)

	triples, err := json.Marshal(testTriples)
	if err != nil {
		t.Error(err)
	}

	buf := bytes.NewBuffer(triples)

	resp, err := http.Post(base+"/api/v1/insert", "application/json", buf)
	if err != nil {
		t.Error(err)
	}
	out, _ := ioutil.ReadAll(resp.Body)
	if !bytes.Contains(out, []byte(strconv.Itoa(len(testTriples)))) {
		t.Errorf("http.Post(/api/v1/insert) = %+v; missing %+v", string(out), len(testTriples))
	}

	var signedTriples []*protocol.Triple

	resp, err = http.Get(base + "/api/v1/triples")
	if err != nil {
		t.Error(err)
	}
	err = json.NewDecoder(resp.Body).Decode(&signedTriples)
	if err != nil {
		t.Error(err)
	}
	if len(signedTriples) != len(testTriples) {
		t.Errorf("http.Get(/api/v1/insert) = %+v; not %+v", signedTriples, testTriples)
	}
}
