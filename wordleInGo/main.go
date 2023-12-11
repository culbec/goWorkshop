package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"github.com/UBBGDSC/gowordleapi/wordle"
)

const lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
const uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const numbers = "0123456789"

// Gets the Wordle preferences for the game
func get(wordUrl string) wordle.WordlePreferences {
	// GET request
	res, err := http.Get(wordUrl)

	if err != nil {
		panic(err)
	}

	var wordlePreferences wordle.WordlePreferences
	err = json.NewDecoder(res.Body).Decode(&wordlePreferences)
	if err != nil {
		panic(err)
	}

	return wordlePreferences
}

// Checks if our guess is the searched word
func post(guess []byte, wordUrl string) wordle.GuessResponse {
	// POST request to validate our guess on the server
	guessRequestBody := wordle.GuessRequest{
		Guess: string(guess),
	}

	jsonData, err := json.Marshal(guessRequestBody)
	if err != nil {
		panic(err)
	}

	res, err := http.Post(wordUrl, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		panic(err)
	}

	if res.StatusCode != http.StatusOK {
		// this means that the guess was not built correctly
		// (length, capital letters, special characters, numbers)
		body, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Error Message: %s\n", string(body))
	}

	var guessResponse wordle.GuessResponse
	err = json.NewDecoder(res.Body).Decode(&guessResponse)
	if err != nil {
		panic(err)
	}

	return guessResponse
}

// Generates a random guess based on the feedback received from the server
func generateGuess(guess []byte, feedback string, wordLength int, charset []byte) []byte {
	for i := 0; i < wordLength; i++ {
		if feedback[i] != '2' {
			// Generating a new random character if the feedback for the character is not valid
			index := rand.Intn(len(charset))
			guess[i] = charset[index]
		}
	}

	return guess
}

// Excludes the characters that are not in the word to guess
func minimizeCharset(charset string, wordLength int, wordUrl string) []byte {
	// Storing all the characters in a map for fast deletion
	charsetMap := make(map[int]byte)
	for index, char := range charset {
		charsetMap[index] = byte(char)
	}

	guess := []byte(strings.Repeat("a", wordLength))

	// Checking which characters are in the word to guess
	for k, v := range charsetMap {
		guess[0] = v

		// Veryfing if the character is in the word to guess
		guessResponse := post(guess, wordUrl)
		if guessResponse.Feedback[0] == '0' {
			// Deleting the character from the charset
			delete(charsetMap, k)
		}
	}

	// Converting the map to a slice that contains only the characters from the valid charset
	charsetSlice := make([]byte, 0, len(charset))
	for _, v := range charsetMap {
		charsetSlice = append(charsetSlice, v)
	}
	return charsetSlice
}

func getCharset(wordlePreferences wordle.WordlePreferences) string {
	// Adding all the lowercase letters to the charset
	charset := lowercaseLetters

	// Adding other characters if they exist
	if wordlePreferences.ContainsCapitalLetters {
		charset += uppercaseLetters
	}
	if wordlePreferences.ContainsNumbers {
		charset += numbers
	}

	return charset
}

// Reveals the easy word from the channel
func revealWord(wordUrl string) {
	// Preferences of the Wordle game
	wordlePreferences := get(wordUrl)

	// Storing the final feedback for fast comparison
	finalFeedback := strings.Repeat("2", wordlePreferences.Length)

	// Storing the current feedback for comparison
	currentFeedback := (strings.Repeat("0", wordlePreferences.Length))

	// Charset of the accepted characters
	charset := getCharset(wordlePreferences)

	// Minimizing the charset so that it contains only the characters in the word to guess
	validCharset := minimizeCharset(charset, wordlePreferences.Length, wordUrl)

	// Number of guesses
	guesses := 0

	// The current guess
	guess := make([]byte, wordlePreferences.Length)

	// Running the algorithm until the feedback is valid
	for currentFeedback != finalFeedback {
		// Generating a new random guess
		guess = generateGuess(guess, currentFeedback, wordlePreferences.Length, validCharset)

		// Trying our guess
		guessResponse := post(guess, wordUrl)
		currentFeedback = guessResponse.Feedback

		// Incrementing the number of guesses
		guesses++
	}

	fmt.Println("Found the word: ", string(guess), ". Total guesses: ", guesses)
}

func main() {
	// HTTP Server structure
	server := &http.Server{
		Addr:    "127.0.0.1:8080", // Change the port as needed
		Handler: http.DefaultServeMux,
	}

	// Ping-Pong Server handler
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// New Wordle Instance
	wordleInstance := wordle.NewWordle()

	// Wordle endpoints
	wordle.SetupServer(server, wordleInstance)

	// Easy words
	go func() {
		for easyWordUrl := range wordleInstance.EasyWordChannel {
			go revealWord(easyWordUrl)
		}
	}()

	// Hard words
	go func() {
		for hardWordUrl := range wordleInstance.HardWordChannel {
			go revealWord(hardWordUrl)
		}
	}()

	// Start the server
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

