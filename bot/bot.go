package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func main() {
	conn, err := net.Dial("tcp", "localhost:3001")
	fmt.Println("Connected!")
	if err != nil {
		log.Fatal(err)
	}
	_, err = conn.Write([]byte("2"))
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(conn)
		for {
			if scanner.Scan() {
				serverMessage := scanner.Text() // Armazena a mensagem do servidor
				fmt.Println("recebi:", serverMessage)
				serverMessage = Reverse(serverMessage)
				fmt.Println("enviei", serverMessage)
				// Envia de volta a mesma mensagem recebida do servidor
				fmt.Fprintln(conn, serverMessage)

			} else if err := scanner.Err(); err != nil {
				log.Println("Error reading from server:", err)
				break // Encerra o loop em caso de erro
			}
		}
		log.Println("done")
		done <- struct{}{} // sinaliza para a gorrotina principal
	}()
	mustCopy(conn, os.Stdin)
	conn.Close()
	<-done // espera a gorrotina terminar
}
