(function (){
	var table = document.querySelector("table")
	var tbody = table.children[0]
	if (tbody.children.length == 0 || tbody.children[0].children.length == 0) {
		table.style.display = 'none'
	}
})()


var pInput = document.querySelector(".person")
var ptr = document.querySelectorAll(".people > tbody > tr")
var pSanta = document.querySelector(".santa")
var table = document.querySelector("table")
var dispsanta = document.querySelector(".dispsanta")

pInput.addEventListener("submit", personListener)
pSanta.addEventListener("click", santaListener)

function personListener(e) {
	e.preventDefault()
	var name = e.target[0].value
	var email = e.target[1].value
	if (name == "" || email == "") {
		return
	}

	table.style.display = 'table'

	pInput.reset()
	var tdName = document.createElement("td")
	tdName.textContent = name
	ptr[0].appendChild(tdName)

	tdName = document.createElement("td")
	tdName.textContent = email
	ptr[1].appendChild(tdName)
}

function Person(id, name, email) {
	this.id = id
	this.name = name
	this.email = email
}

function getPerson(col) {
	var p = new Person()
	p.id = col
	p.name = ptr[0].children[col].textContent
	p.email = ptr[1].children[col].textContent
	return p
}

function postListener(e) {
	dispsanta.textContent = e.target.getResponseHeader("Santa-Mail-Status")
}

function getpath(location) {
	if (window.location.href[window.location.href.length-1] != "/") {
		return window.location.href + "/" + location
	}
	return window.location.href + location
}

function santaListener() {
	var columns = ptr[0].children.length
	var people = []
	for (var c = 0; c < columns; c++) {
		people[c] = getPerson(c)
	}

	var post = new XMLHttpRequest()
	post.addEventListener("load", postListener)
	post.open("POST", getpath("post/"))
	post.setRequestHeader('Content-Type', 'application/json; charset=utf-8')
	post.send(JSON.stringify(people))
}