package daemon

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/common/whitelabeldelete"
)

func (d *Daemon) sweepWhitelabel() {
	query := `SELECT "user_id" FROM whitelabel;`
	rows, err := d.db.Whitelabel.Query(context.Background(), query)
	defer rows.Close()

	if err != nil {
		sentry.Error(err)
		return
	}

	for rows.Next() {
		var userId uint64
		if err := rows.Scan(&userId); err != nil {
			sentry.Error(err)
			continue
		}

		hasWhitelabel, err := d.hasWhitelabel(userId)
		if err != nil {
			sentry.Error(err)
			return
		}

		if !hasWhitelabel {
			// get bot ID
			bot, err := d.db.Whitelabel.GetByUserId(userId)
			if err != nil {
				sentry.Error(err)
				return
			}

			fmt.Printf("whitelabel: deleting %d (%d)\n", bot.BotId, bot.UserId)

			if err := d.db.Whitelabel.Delete(userId); err != nil {
				sentry.Error(err)
				return
			}

			whitelabeldelete.Publish(d.redis, bot.BotId)
		}
	}
}

// use our own function w/ error handling
func (d *Daemon) hasWhitelabel(userId uint64) (bool, error) {
	fmt.Println(1)
	hasWhitelabelKey, err := d.db.WhitelabelUsers.IsPremium(userId)
	fmt.Println(2)
	if err != nil {
		return false, err
	}
	fmt.Println(3)

	if hasWhitelabelKey {
		return true, nil
	}
	fmt.Println(4)

	tier, err := d.patreon.GetTier(userId)
	fmt.Println(5)
	if err != nil {
		return false, err
	}
	fmt.Println(6)

	if tier >= premium.Whitelabel {
		return true, nil
	}

	fmt.Println(7)
	for _, forced := range d.forced {
		if forced == userId {
			return true, nil
		}
	}

	fmt.Println(8)

	return false, nil
}
