package daemon

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/jackc/pgx/v4"
)

const freePanelLimit = 3

func (d *Daemon) sweepPanels() {
	query := `SELECT "guild_id", COUNT(*) FROM panels GROUP BY guild_id HAVING COUNT(*) > $1;`
	rows, err := d.db.Panel.Query(context.Background(), query, freePanelLimit)
	defer rows.Close()
	if err != nil {
		sentry.Error(err)
		return
	}

	guilds := make(map[uint64]int)
	for rows.Next() {
		var guildId uint64
		var panelCount int
		if err := rows.Scan(&guildId, &panelCount); err != nil {
			sentry.Error(err)
			continue
		}

		guilds[guildId] = panelCount
	}

	fmt.Printf("Detected %d guilds with > 1 panel\n", len(guilds))

	batch := &pgx.Batch{}

	for guildId, panelCount := range guilds {
		// get guild owner
		guild, success := d.cache.GetGuild(guildId, false)
		if !success || guild.OwnerId == 0 { // if bot's been kicked doesn't matter, when we rejoin we'll purge
			continue
		}

		// TODO: Ignore voting?
		tier, _, err := d.premiumClient.GetTierByGuild(guild)
		if err != nil {
			fmt.Printf("error getting premium status for guild %d: %s", guild.Id, err.Error())
			sentry.Error(err)
			continue
		}

		if tier < premium.Premium {
			fmt.Printf("guild %d (owner: %d) is not a patron anymore! panel count: %d\n", guildId, guild.OwnerId, panelCount)

			query := `
				DELETE FROM
					panels
				WHERE
					"message_id" IN (
						SELECT
							"message_id"
						FROM
							panels
						WHERE
							"guild_id" = $1
						LIMIT $2
					)
				;
				`

			batch.Queue(query, guildId, panelCount-freePanelLimit)
		}
	}

	fmt.Println(batch.Len())

	/*if _, err := d.db.Panel.SendBatch(context.Background(), batch).Exec(); err != nil {
		sentry.Error(err)
	}*/
	fmt.Println("done panels")
}
