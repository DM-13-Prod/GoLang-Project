package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"todo/internal/grpcapi"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial(
    "127.0.0.1:50505",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := grpcapi.NewTodoServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Создание задачи
	createRes, err := client.Create(ctx, &grpcapi.CreateTaskRequest{
		Title:       "gRPC Example",
		Description: "Created over gRPC",
		Priority:    2,
	})
	if err != nil {
		log.Fatal("Create:", err)
	}
	fmt.Println("Created task ID:", createRes.Id)

	// Получаем по id
	task, err := client.Get(ctx, &grpcapi.TaskID{Id: createRes.Id})
	if err != nil {
		log.Fatal("Get:", err)
	}
	fmt.Println("Fetched:", task)

	// Обновляем
	_, err = client.Update(ctx, &grpcapi.UpdateTaskRequest{
		Id:       createRes.Id,
		Status:   "done",
		Priority: 3,
	})
	if err != nil {
		log.Println("Update:", err)
	}

	// Получаем список
	list, err := client.List(ctx, &grpcapi.Empty{})
	if err != nil {
		log.Println("List:", err)
	}
	for _, item := range list.Items {
		fmt.Printf("→ %v [%v]\n", item.Title, item.Status)
	}

	// Удаляем
	_, err = client.Delete(ctx, &grpcapi.TaskID{Id: createRes.Id})
	if err != nil {
		log.Println("Delete:", err)
	} else {
		fmt.Println("Deleted", createRes.Id)
	}
}