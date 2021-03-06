package function

import (
	"time"
	"fmt"
	"tarot/model"
	"tarot/util"
	"tarot/plugins/tarot"
)

func sendHandler(event *model.TarotEvent) {
	util.Info(`try to send event ` + event.NickName + event.SentenceType)
	//for model.ApplicationEvents.GetUnNilAmount() > 50 {
	//	util.SocketInfo(fmt.Sprintf(`more than 2 events in nickEvents, sleep 3 seconds`))
	//	time.Sleep(time.Second * 3)
	//}
	if event == nil {
		return
	}
	if event.FromTarotStatus != event.ToTarotStatus {
		util.Info(`save db event ` + event.NickName + event.SentenceType)
		util.Info(fmt.Sprintf(`%s update status from %d to %d`, event.NickName, event.FromTarotStatus,
			event.ToTarotStatus))
		model.DB.Model(&model.MyContact{}).Where(`nick_name=?`, event.NickName).
			Updates(map[string]interface{}{`tarot_status`: event.ToTarotStatus, `updated_at`: time.Now()})
	}
	bytes := []byte(event.ToUserName)
	if bytes[0] == '@' && bytes[1] == '@' { //过滤掉@@开头的userName(微信群)
		return
	}
	util.SendTarotMsg(event.NickName, event.FromUserName, event.ToUserName, event.SentenceType)
	model.ApplicationEvents.RemoveEvent(event.NickName)
}

func PlayTarot() {
	for true {
		rows, err := model.DB.Table("my_contacts").Select("nick_name,tarot_status,updated_at").
			Where("tarot_nick_name = ?", model.AppBot.Bot.NickName).Rows()
		if err != nil {
			util.Notice(`db error:` + fmt.Sprint(err))
			continue
		}
		for rows.Next() {
			var nickName string
			var tarotStatus int
			var updatedAt time.Time
			rows.Scan(&nickName, &tarotStatus, &updatedAt)
			toTarotStatus := tarot.CheckTime(tarotStatus, updatedAt)
			if toTarotStatus == 0 {
				continue
			}
			contacts := model.AppBot.Cm.GetContactsByName(nickName)
			for _, value := range contacts {
				sentenceType := fmt.Sprintf(`%d-%d`, tarotStatus, toTarotStatus)
				event := model.TarotEvent{FromUserName: model.AppBot.Bot.UserName, ToUserName: value.UserName,
					SentenceType: sentenceType, NickName: nickName, FromTarotStatus: tarotStatus, ToTarotStatus: toTarotStatus}
				util.Info(fmt.Sprintf("%s play tarot with sentence %s", nickName, sentenceType))
				model.SendChannel <- event
			}
		}
		time.Sleep(time.Second * 10)
	}
}

func SendChannelServe() {
	for true {
		event := <-model.SendChannel
		util.Info(fmt.Sprintf(`get event of %s from channel with %s from %d to %d`,
			event.NickName, event.SentenceType, event.FromTarotStatus, event.ToTarotStatus))
		currentEvent := model.ApplicationEvents.GetEvent(event.NickName)
		if currentEvent != nil {
			util.SocketInfo(event.NickName + `old event ` + currentEvent.SentenceType + ` abandon ` + event.SentenceType)
			continue
		}
		model.ApplicationEvents.PutEvent(event.NickName, &event)
		go sendHandler(&event)
	}
}
