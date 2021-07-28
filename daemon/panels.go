package daemon

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/rxdn/gdl/objects/guild"
	"github.com/rxdn/gdl/objects/guild/emoji"
	"github.com/rxdn/gdl/objects/interaction/component"
	"github.com/rxdn/gdl/rest"
	"github.com/rxdn/gdl/utils"
	"os"
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

	d.Logger.Printf("Detected %d guilds with > 1 panel\n", len(guilds))

	var ok, notOk int

	for guildId, panelCount := range guilds {
		// get guild owner
		guild, success := d.cache.GetGuild(guildId, false)
		if !success || guild.OwnerId == 0 { // if bot's been kicked doesn't matter, when we rejoin we'll purge
			continue
		}

		// TODO: Ignore voting?
		tier, _, err := d.premiumClient.GetTierByGuild(guild)
		if err != nil {
			d.Logger.Printf("error getting premium status for guild %d: %s", guild.Id, err.Error())
			sentry.Error(err)
			continue
		}

		if tier < premium.Premium {
			notOk++
			d.Logger.Printf("guild %d (owner: %d) is not a patron anymore! panel count: %d (%d)\n", guildId, guild.OwnerId, panelCount, notOk)

			// Delete with select subquery destroys CPU
			// Instead, select X-3 panels first
			panels, err := d.db.Panel.GetByGuild(guildId)
			if err != nil {
				d.Logger.Printf("error getting panels for guild %d: %s", guild.Id, err.Error())
				sentry.Error(err)
				continue
			}

			// Double check
			if len(panels) < 3 {
				continue
			}

			if !d.dryRun {
				// TODO: Bulk
				for i := 0; i < len(panels)-3; i++ {
					if err := d.db.Panel.Delete(panels[i].PanelId); err != nil {
						d.Logger.Printf("error deleting panels for guild %d: %s", guild.Id, err.Error())
						sentry.Error(err)
						continue
					}
				}
			}
		} else {
			ok++
			d.Logger.Printf("guild %d (owner: %d) is ok (%d)\n", guildId, guild.OwnerId, ok)
		}
	}

	d.Logger.Printf("done panels")
}

func warn(ownerId uint64, guild guild.Guild) {
	token := os.Getenv("BOT_TOKEN")

	ch, err := rest.CreateDM(token, nil, ownerId)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	content := fmt.Sprintf(":warning: Your server `%s` has exceeded the free panel quota (3). As a result, the additional panels will be deleted. If you believe this is in error, please join our support server by clicking the button below.", guild.Name)
	data := rest.CreateMessageData{
		Content: content,
		Components: []component.Component{
			component.BuildActionRow(component.BuildButton(component.Button{
				Label: "Support Server",
				Style: component.ButtonStyleLink,
				Emoji: emoji.Emoji{
					Name: "👋",
				},
				Url: utils.StrPtr("https://discord.gg/VtV3rSk"),
			})),
		},
	}

	_, err = rest.CreateMessage(token, nil, ch.Id, data)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("warned %d\n", ownerId)
}
