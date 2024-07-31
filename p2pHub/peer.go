package main

type AnswerPeer struct{}

func (ap *AnswerPeer) serveAnswer() {

}

func servePeer() {
	answerPeer := &AnswerPeer{}

	go answerPeer.serveAnswer()
}