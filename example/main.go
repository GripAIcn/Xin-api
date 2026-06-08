package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 初始化模型
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  "sk-xin-0f068cff29f0f62a3caa1d6c43ec2927",
		Model:   "deepseek-v4-flash",
		BaseURL: "http://api.gripai.cn/v1",
	})
	if err != nil {
		log.Fatal(err)
	}

	msg := []*schema.Message{
		&schema.Message{Role: schema.User, Content: "你是什么模型？"},
	}

	out, err := model.Generate(context.Background(), msg)

	if err != nil {
		fmt.Printf("AI: %s\n", err.Error())
	}

	fmt.Printf("AI SUCCEES: %s\n", out)
}
