package daemon

import (
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/go-redis/redis"
	"github.com/rxdn/gdl/cache"
	"os"
	"strconv"
	"strings"
	"time"
)

type Daemon struct {
	db      *database.Database
	cache   *cache.PgCache
	redis   *redis.Client
	patreon *premium.PatreonClient
	forced  []uint64
}

func NewDaemon(db *database.Database, cache *cache.PgCache, redis *redis.Client, patreon *premium.PatreonClient) *Daemon {
	var forced []uint64
	for _, raw := range strings.Split(os.Getenv("FORCED"), ",") {
		if raw == "" {
			continue
		}

		userId, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			sentry.Error(err)
			continue
		}

		forced = append(forced, userId)
	}

	return &Daemon{
		db:      db,
		cache:   cache,
		redis:   redis,
		patreon: patreon,
		forced:  forced,
	}
}

func (d *Daemon) Start() {
	for {
		d.sweepWhitelabel()
		d.sweepPanels()
		time.Sleep(time.Hour * 6)
	}
}
