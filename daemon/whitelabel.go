package daemon

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/common/whitelabeldelete"
)

func (d *Daemon) sweepWhitelabel() {
	fmt.Println(1)
	query := `SELECT "user_id" FROM whitelabel;`
	rows, err := d.db.Whitelabel.Query(context.Background(), query)
	fmt.Println(2)
	defer rows.Close()

	if err != nil {
		sentry.Error(err)
		return
	}
	fmt.Println(3)

	for rows.Next() {
		fmt.Println(4)
		var userId uint64
		if err := rows.Scan(&userId); err != nil {
			sentry.Error(err)
			continue
		}
		fmt.Println(5)

		hasWhitelabel, err := d.hasWhitelabel(userId)
		if err != nil {
			sentry.Error(err)
			return
		}
		fmt.Println(6)

		if !hasWhitelabel {
			fmt.Println(7)
			// get bot ID
			bot, err := d.db.Whitelabel.GetByUserId(userId)
			if err != nil {
				sentry.Error(err)
				return
			}
			fmt.Println(8)

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
	hasWhitelabelKey, err := d.db.WhitelabelUsers.IsPremium(userId)
	if err != nil {
		return false, err
	}

	if hasWhitelabelKey {
		return true, nil
	}

	tier, err := d.patreon.GetTier(userId)
	if err != nil {
		return false, err
	}

	if tier >= premium.Whitelabel {
		return true, nil
	}

	for _, forced := range d.forced {
		if forced == userId {
			return true, nil
		}
	}

	return false, nil
}
