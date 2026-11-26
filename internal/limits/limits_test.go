package limits

import (
	"context"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/foxcpp/maddy/framework/config"
	"github.com/foxcpp/maddy/internal/testutils"
)

func TestTakeMsg(t *testing.T) {
	for _, scope := range []string{"all", "ip", "source"} {
		synctest.Test(t, func(t *testing.T) {
			_TestTakeMsgSync(t, scope)
		})
	}
}

func _TestTakeMsgSync(t *testing.T, scope string) {
	t.Logf("[%s] TakeMsg test starting", scope)
	limits := Empty(testutils.Logger(t, "limits"))
	err := limits.Configure(nil, config.NewMap(nil, config.Node{
		Children: []config.Node{
			{
				Name: scope,
				Args: []string{"rate", "2"},
			},
			{
				Name: scope,
				Args: []string{"concurrency", "6"},
			},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	var (
		releases = make([]ReleaseFunc, 6)
		addr     = net.IPv4(127, 0, 0, 1)
		domain   = "example1.invalid"
	)

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	for i := range 6 {
		r, err := limits.TakeMsg(ctx, addr, domain)
		if err != nil {
			t.Fatalf("TakeMsg: %v", err)
		}
		releases[i] = r
	}
	if time.Since(start) != time.Second*2 {
		t.Fatalf("Excepted taking 2s, but took %s", time.Since(start))
	}

	if _, err := limits.TakeMsg(ctx, addr, domain); err == nil {
		t.Fatal("Excepted a limits error, but got none")
	}
	releases[0]()

	// ctx already timeout, need a new Context
	nctx, ncancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer ncancel()
	if _, err := limits.TakeMsg(nctx, addr, domain); err != nil {
		t.Fatal(err)
	}

	// test different ip/domain
	release, err := limits.TakeMsg(nctx, net.IPv4(127, 0, 0, 2), "example2.invalid")
	if scope == "all" && err == nil {
		// global limit shoud limit all ip and destination
		t.Fatal("Excepted a limits error, but got none")
	} else if scope != "all" && err != nil {
		// per-ip/per-domain limits shouldn't timeout
		t.Fatal(err)
	}

	if err == nil {
		release()
	}

	// release again, take no effort
	releases[0]()
	if _, err := limits.TakeMsg(nctx, addr, domain); err == nil {
		t.Fatal("Excepted a limits error, but got none")
	}

	t.Logf("[%s] TakeMsg test took %s", scope, time.Since(start))
}

func TestTakeDest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Log("TakeDest test starting")
		limits := Empty(testutils.Logger(t, "limits"))
		err := limits.Configure(nil, config.NewMap(nil, config.Node{
			Children: []config.Node{
				{
					Name: "destination",
					Args: []string{"rate", "2"},
				},
				{
					Name: "destination",
					Args: []string{"concurrency", "6"},
				},
			},
		}))
		if err != nil {
			t.Fatal(err)
		}
		var (
			releases = make([]ReleaseFunc, 6)
			domain   = "example.invalid"
		)

		start := time.Now()
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
		defer cancel()

		for i := range 6 {
			r, err := limits.TakeDest(ctx, domain)
			if err != nil {
				t.Fatalf("TakeDest: %v", err)
			}
			releases[i] = r
		}

		if time.Since(start) != time.Second*2 {
			t.Fatalf("Excepted taking 2s, but took %s", time.Since(start))
		}

		if _, err := limits.TakeDest(ctx, domain); err == nil {
			t.Fatal("Excepted a limits error, but got none")
		}

		releases[0]()

		// ctx already timeout, need a new Context
		nctx, ncancel := context.WithTimeout(context.TODO(), time.Second*3)
		defer ncancel()
		if _, err := limits.TakeDest(nctx, domain); err != nil {
			t.Fatal(err)
		}

		// release again, take no effort
		releases[0]()

		if _, err := limits.TakeDest(nctx, domain); err == nil {
			t.Fatal("Excepted a limits error, but got none")
		}

		t.Logf("TakeDest test took %s", time.Since(start))
	})
}
