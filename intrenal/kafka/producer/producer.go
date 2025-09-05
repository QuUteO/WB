package producer

import "github.com/IBM/sarama"

func NewCfgProducer() {
	NewCfg := sarama.NewConfig()
	NewCfg.Producer.RequiredAcks = sarama.WaitForAll
	NewCfg.Producer.Retry.Max = 5
	NewCfg.Producer.Return.Successes = true
	NewCfg.Producer.Return.Errors = true
}
