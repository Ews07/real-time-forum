let messagesOffset = 0
let chatWith = ""; // user UUID you're chatting with
let loading = false
const chatHistory = document.getElementById("chat-history")

function loadMessages() {
  if (loading) return
  loading = true

  fetch(`/messages?with=${chatWith}&offset=${messagesOffset}`, {
    method: "GET",
    credentials: "include",
  })
    .then((res) => res.json())
    .then((messages) => {
      messagesOffset += messages.length

      const oldHeight = chatHistory.scrollHeight

      messages.forEach((msg) => {
        const div = document.createElement("div")
        div.textContent = `${msg.from}: ${msg.content}`
        chatHistory.prepend(div)
      })

      // Restore scroll position
      chatHistory.scrollTop = chatHistory.scrollHeight - oldHeight

      loading = false;
    })
}


let debounceTimer

chatHistory.addEventListener("scroll", function () {
  if (chatHistory.scrollTop === 0) {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      loadMessages();
    }, 300) // 300ms debounce
  }
})

function openChat(userUUID) {
  chatWith = userUUID
  messagesOffset = 0
  chatHistory.innerHTML = ""
  loadMessages() // Load first 10 messages
}