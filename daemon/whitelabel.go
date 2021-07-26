package daemon

import (
	"context"
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
		d.Logger.Printf("error getting whitelabel users: %s", err.Error())
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
			d.Logger.Printf("error checking whitelabel for %d: %s", userId, err.Error())
			return
		}

		if !hasWhitelabel {
			// get bot ID
			bot, err := d.db.Whitelabel.GetByUserId(userId)
			if err != nil {
				sentry.Error(err)
				return
			}

			d.Logger.Printf("whitelabel: deleting %d (%d)\n", bot.BotId, bot.UserId)

			if err := d.db.Whitelabel.Delete(userId); err != nil {
				sentry.Error(err)
				d.Logger.Printf("error deleting whitelabel for %d: %s", userId, err.Error())
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
