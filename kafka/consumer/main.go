package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Shopify/sarama"
	"log"
	//"logger"
	//"time"
)

type exampleConsumerGroupHandler struct {
	x *XXData
}

func (exampleConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (exampleConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h exampleConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		res := fmt.Sprintf("Message topic:%q partition:%d offset:%d value:%s\n", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		h.x.SendData(res)
		sess.MarkMessage(msg, "")
	}
	return nil
}

var consumerGroup = "1"

func main() {
	flag.StringVar(&consumerGroup, "c", "1", "test")
	flag.Parse()
	fmt.Printf("consumerGroup:%s\n", consumerGroup)
	kfversion, err := sarama.ParseKafkaVersion("0.11.0.2") // kafkaVersion is the version of kafka app like 0.11.0.2
	if err != nil {
		log.Println(err)
	}

	config := sarama.NewConfig()
	config.Version = kfversion
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.AutoCommit.Enable = true

	// Start with a client
	client, err := sarama.NewClient([]string{"noah-machine:9092"}, config)
	if err != nil {
		log.Println(err)
	}
	defer func() { _ = client.Close() }()

	// Start a new consumer group
	group, err := sarama.NewConsumerGroupFromClient(consumerGroup, client)
	if err != nil {
		log.Println(err)
	}
	defer func() { _ = group.Close() }()

	// Track errors
	go func() {
		for err := range group.Errors() {
			fmt.Println("ERROR", err)
		}
	}()

	done := make(chan error, 2)
	stop := make(chan struct{})

	t := NewXXData(5, 5)
	go func() {
		t.Run(stop)
		done <- nil
	}()

	// Iterate over consumer sessions.
	ctx := context.Background()
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				topics := []string{"test_log"}
				handler := exampleConsumerGroupHandler{x: t}

				// `Consume` should be called inside an infinite loop, when a
				// app-side rebalance happens, the consumer session will need to be
				// recreated to get the new claims
				err := group.Consume(ctx, topics, handler)
				if err != nil {
					//panic(err)
					done <- err
				}
			}
		}
	}()

	fmt.Println("consumer end")
	var stoped bool
	for i := 0; i < cap(done); i++ {
		err := <-done
		if err != nil {
			//记录错误信息
			log.Println(err)
		}
		if !stoped {
			stoped = true
			close(stop)
		}
	}

	fmt.Println("consumer end end")

	/*	consumer, err := sarama.NewConsumer([]string{"10.12.237.171:9092","10.12.237.171:9093","10.12.237.171:9094"}, nil)
		if err != nil {
			fmt.Printf("fail to start consumer, err:%v\n", err)
			return
		}
		partitionList, err := consumer.Partitions("test_log") // 根据topic取到所有的分区
		if err != nil {
			fmt.Printf("fail to get list of partition:err%v\n", err)
			return
		}
		fmt.Println(partitionList)

		defer consumer.Close()
		for partition := range partitionList { // 遍历所有的分区
			fmt.Printf("partion:%d\n", partition)
			// 针对每个分区创建一个对应的分区消费者
			pc, err := consumer.ConsumePartition("test_log", int32(partition), sarama.OffsetNewest)
			if err != nil {
				fmt.Printf("failed to start consumer for partition %d,err:%v\n", partition, err)
				return
			}

			//同步消费信息
			func(pc sarama.PartitionConsumer) {
				defer pc.Close()

				for message := range pc.Messages() {
					logger.Printf("[Consumer] partitionid: %d; offset:%d, value: %s\n", message.Partition, message.Offset, string(message.Value))
				}
			}(pc)

			//defer pc.AsyncClose()
			//// 异步从每个分区消费信息
			//go func(pc sarama.PartitionConsumer) {
			//	for msg := range pc.Messages() {
			//		fmt.Printf("Partition:%d Offset:%d Key:%v Value:%s", msg.Partition, msg.Offset, msg.Key, msg.Value)
			//	}
			//}(pc)
		}
		time.Sleep(10 * time.Second)*/
}
