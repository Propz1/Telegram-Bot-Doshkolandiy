package types

type BotMessage struct {
	Message struct {
		Message_id int
		From       struct {
			Username string
			Id       int
		}
		Chat struct {
			Id int
		}
		Text string
	}
}

type BotSendMessgeID struct {
	Result struct {
		Message_id int
	}
}

type Photos struct {
	Entries []struct {
		FullPath string
		Mime     string
	}
}
