package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

type client chan<- string // Canal para envio de mensagens ao cliente

// Estrutura para representar uma pessoa
type Person struct {
	nome  string // Nome do usuário
	tipo  string // Tipo de usuário (cliente ou bot)
	porta client // Canal de mensagens do cliente
}

var (
	entering     = make(chan client)         // Canal de entrada para novos clientes
	leaving      = make(chan client)         // Canal para clientes que saem
	messages     = make(chan string)         // Canal para mensagens gerais
	pessoas      = make(map[net.Conn]Person) // Mapeia conexões para pessoas
	conns        []net.Conn                  // Lista de conexões ativas
	numberOfBots = 0                         // Contador de bots conectados
)

// Goroutine que distribui mensagens a todos os clientes
func broadcaster() {
	clients := make(map[client]bool) // Mapeia todos os clientes conectados
	for {
		select {
		case msg := <-messages:
			// Envia mensagem para todos os clientes não-bots
			for _, conn := range conns {
				if pessoas[conn].tipo != "bot" {
					pessoas[conn].porta <- msg
				}
			}
		case cli := <-entering:
			clients[cli] = true
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

// Encontra cliente pelo nome
func findClient(nome string) net.Conn {
	nome = strings.TrimSpace(nome)
	for conn, storedPerson := range pessoas {
		if nome == storedPerson.nome {
			return conn
		}
	}
	return nil
}

// Verifica se um nome já existe entre os clientes
func nameExist(nome string) bool {
	for _, valor := range pessoas {
		if valor.nome == nome {
			return true
		}
	}
	return false
}

// Envia mensagem privada a outro usuário
func privateMessage(restante string, conn net.Conn, ch chan string) {
	restante = strings.TrimSpace(restante)
	indice := strings.Index(restante, " ")
	Pch := make(chan string)

	if indice != -1 {
		mensagem := restante[indice+1:]
		destinatario := strings.TrimSpace(restante[:indice])
		connDestino := findClient(destinatario)

		if connDestino != nil {
			go clientWriter(connDestino, Pch)
			Pch <- pessoas[conn].nome + " -> " + pessoas[connDestino].nome + ": " + mensagem
			ch <- pessoas[conn].nome + " -> " + pessoas[connDestino].nome + ": " + mensagem
			close(Pch)
		} else {
			ch <- "Destinatário não encontrado"
		}
	} else {
		ch <- " Mensagem inválida"
	}

}

// Envia mensagem para um bot
func commandBot(restante string, ch chan string) {
	restante = strings.TrimSpace(restante)
	indice := strings.Index(restante, " ")
	Bch := make(chan string)

	if indice != -1 {
		mensagem := restante[indice+1:]
		destinatario := strings.TrimSpace(restante[:indice])
		connDestino := findClient(destinatario)

		if connDestino != nil {
			if pessoas[connDestino].tipo != "bot" {
				ch <- "o usuario nao é bot"
				return
			}
			go clientWriter(connDestino, Bch)
			Bch <- mensagem

			scanner := bufio.NewScanner(connDestino)
			if scanner.Scan() {
				messages <- "bot " + pessoas[connDestino].nome + ": " + scanner.Text()
				close(Bch)
			} else {
				ch <- "Erro no scanner"
			}

		} else {
			ch <- "Bot nao encontrado"
		}
	} else {
		ch <- "Mensagem Invalida"
	}
}

// Goroutine que envia mensagens para o cliente
func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

// Manipula uma conexão de cliente
func handleConn(conn net.Conn) {
	ch := make(chan string)
	go clientWriter(conn, ch)

	var pessoa Person
	pessoa.porta = ch

	// Lê o tipo de usuário (cliente ou bot)
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	// Verifica tipo de conexão (cliente ou bot) e registra nome
	if string(buffer[:n]) == "1" {
		pessoa.tipo = "client"
		ch <- "Qual seu nome?:"
		nome := bufio.NewScanner(conn)
		for nome.Scan() {
			if nameExist(nome.Text()) {
				ch <- "Esse Nick ja esta sendo usado\nescreva um nome:"
			} else {
				pessoa.nome = strings.TrimSpace(nome.Text())
				break
			}
		}
	} else if string(buffer[:n]) == "2" {
		numberOfBots++
		pessoa.tipo = "bot"
		pessoa.nome = "superBot" + strconv.Itoa(numberOfBots)
	}

	pessoas[conn] = pessoa
	if pessoa.tipo != "bot" {
		ch <- "vc é " + pessoa.nome
	}

	messages <- pessoa.tipo + " @" + pessoa.nome + " chegou!"
	entering <- ch

	// Loop para receber e processar comandos dos clientes
	input := bufio.NewScanner(conn)
	for input.Scan() {
		if input.Text() == "/sair" {
			break
		} else if strings.HasPrefix(input.Text(), "/cn") {
			go func() {
				restante := strings.TrimSpace(input.Text()[len("/cn "):])
				if restante == "" {
					ch <- "coloque um nome valido "
				} else if nameExist(restante) {
					ch <- "nome ja sendo usado"
				} else {
					apelidoAntigo := pessoa.nome
					pessoa := pessoas[conn]
					pessoa.nome = restante
					pessoas[conn] = pessoa
					messages <- apelidoAntigo + " virou " + pessoa.nome
				}
			}()

		} else if strings.HasPrefix(input.Text(), "/msg") {
			restante := input.Text()[len("/msg "):]
			go privateMessage(restante, conn, ch)
		} else if strings.HasPrefix(input.Text(), "/bot") {
			restante := input.Text()[len("/bot "):]
			go commandBot(restante, ch)

		} else {
			messages <- pessoa.nome + ": " + input.Text()
		}
	}

	leaving <- ch
	messages <- pessoa.nome + " se foi "
	conn.Close()
}

// Função principal do servidor de chat
func main() {
	fmt.Println("Iniciando servidor...")
	listener, err := net.Listen("tcp", "localhost:3001")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("servidor conectado")
	}

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		conns = append(conns, conn)
		go handleConn(conn)
	}
}
