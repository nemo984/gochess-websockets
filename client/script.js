var board = null;
var game = new Chess();
var $status = $("#status");
var $myColor = $('#myColor')
var $fen = $("#fen");
var $pgn = $("#pgn");
var $gameID = $("#game_id");
var inGame = false;

//Start off with white color
var myColor = "w";

//sfx
var captureSfx = new Audio("./sounds/Capture.ogg");
var moveSfx = new Audio("./sounds/Move.ogg");

function playSfx(sfx) {
	playPromise = sfx.play();
	if (playPromise !== undefined) {
		playPromise
			.then(function () { })
			.catch(function (error) {
				console.error("Can't play sfx:", error.message);
			});
	}
}

function onDragStart(source, piece, position, orientation) {
	if (game.game_over()) return false;
	c1 = inGame ? myColor : "b";
	c2 = inGame ? myColor : "w";
	if (
		(game.turn() !== c1 && piece.search(/^b/) !== -1) ||
		(game.turn() !== c2 && piece.search(/^w/) !== -1)
	) {
		return false;
	}
}

function onDrop(source, target) {
	var move = game.move({
		from: source,
		to: target,
		promotion: "q", // NOTE: always promote to a queen
	});

	// illegal move
	if (move === null) return "snapback";

	if (move.san.indexOf("x") > -1) {
		playSfx(captureSfx);
	} else {
		playSfx(moveSfx);
	}
	updateStatus();
	if (inGame) {
		if (ws.readyState === WebSocket.CLOSED) {
			alert("Websocket is closed");
		}
		ws.send(
			JSON.stringify({
				action: "move",
				data: {
					id: $gameID.val(),
					move: move.san,
				},
			})
		);
	}
}

function onSnapEnd() {
	board.position(game.fen());
}

function updateStatus() {
	var status = "";

	var moveColor = "White";
	if (game.turn() === "b") {
		moveColor = "Black";
	}

	if (game.in_checkmate()) {
		status = "Game over, " + moveColor + " is in checkmate.";
	} else if (game.in_draw()) {
		status = "Game over, drawn position";
	} else {
		status = moveColor + " to move";

		if (game.in_check()) {
			status += ", " + moveColor + " is in check";
		}
	}

	$myColor.html(myColor === 'w' ? 'White' : 'Black')
	$status.html(status);
	$fen.html(game.fen());
	$pgn.html(game.pgn());
}



const servers = {
	iceServers: [
		{
			urls: ['stun:stun1.l.google.com:19302', "stun:stun2.l.google.com:19302"],
		},
	],
	iceCandidatePoolSize: 10
}

const webcamVideo = document.getElementById('webcamVideo')
const remoteVideo = document.getElementById('remoteVideo')

let pc = new RTCPeerConnection(servers)
let localStream = null;
let remoteStream = null;

(async () => {
	localStream = await navigator.mediaDevices.getUserMedia({
		video: true,
		audio: true
	})
	remoteStream = new MediaStream()

	localStream.getTracks().forEach((track) => {
		pc.addTrack(track, localStream)
	})

	// tracks (video/audio) from remote stream 
	pc.ontrack = event => {
		event.streams[0].getTracks().forEach(track => {
			remoteStream.addTrack(track)
		})
	}

	webcamVideo.srcObject = localStream
	remoteVideo.srcObject = remoteStream
})();

var config = {
	draggable: true,
	position: "start",
	onDragStart: onDragStart,
	onDrop: onDrop,
	onSnapEnd: onSnapEnd,
};
board = Chessboard("board", config);
updateStatus();

host = window.location.hostname + ":" + window.location.port;
ws = new WebSocket(`ws://${host}/ws`);

function joinGame(id) {
	if (ws.readyState === WebSocket.CLOSED || ws.readyState == WebSocket.CLOSIN) {
		alert("Websocket is closed");
		ws = new WebSocket(`ws://${host}/ws`);
	}
	if (id === "") {
		alert("GAME ID INPUT IS EMPTY");
	}

	const answerDescription = ""
	pc.onicecandidate = event => {
		console.log(event.candidate.toJSON())
		if (event.candidate) {
			ws.send(JSON.stringify({
				action: "offer",
				candidate: event.candidate.toJSON()
			}))
		}
	}


	pc.createAnswer()
		.then(answer => {
			const answerDescription = answer
			pc.setLocalDescription(answerDescription)
				.then(() => {
					const answer = {
						sdp: answerDescription.sdp,
						type: answerDescription.type
					}
					console.log(answer)
					ws.send(
						JSON.stringify({
							action: "join",
							data: {
								id: id,
								sdp: answer,
							},
						})
					);
				})
		})

}

ws.onopen = (e) => {
	const gameID = sessionStorage.getItem("id");
	if (gameID !== null) {
		joinGame(gameID);
	}
};

ws.onmessage = (e) => {
	json = JSON.parse(e.data);
	console.log(json);

	if (json.hasOwnProperty("answer")) {
		const data = json.answer
		if (!pc.currentRemoteDescription && data?.answer) {
			const answerDescription = new RTCSessionDescription(data.answer);
			pc.setRemoteDescription(answerDescription)
		}
		return
	}


	if (json.hasOwnProperty("pgn") && json.hasOwnProperty("id")) {
		sessionStorage.setItem("id", json.id);

		inGame = true;
		$gameID.html(json.id);
		game.load_pgn(json.pgn);
		board.position(json.fen, false);
		updateStatus();
		//make better event
		switch (json.event) {
			case ("Game Created", "Game joined as white"):
				myColor = "w";
				break;

			case "Game joined as black":
				myColor = "b";
				board.flip();
				break;

			case "Player join game":
				if (!pc.currentRemoteDescription && json?.answer) {
					const answerDescription = new RTCSessionDescription(json.answer)
					pc.setRemoteDescription(answerDescription)
				}
				break;

			case "ICE":
				const candidate = new RTCIceCandidate(json.data)
				pc.addIceCandidate(candidate)
				return
		}
		//Default: move event
		//TODO: maybe also send back game status & color that play the move
		//Sound effects
		if (
			json.event !== "Game Created" &&
			json.event !== "Player join game"
		) {
			if (json.event.split(" ")[0].indexOf("x") > -1) {
				playSfx(captureSfx);
			} else {
				playSfx(moveSfx);
			}
		}
	}

	if (json.hasOwnProperty("message")) {
		alert(json.message);
		sessionStorage.clear();
	}
};

$("#flipOrientationBtn").on("click", board.flip);

$(".createBtn").on("click", async () => {
	console.log(ws.readyState)
	if (ws.readyState === WebSocket.CLOSED || ws.readyState == WebSocket.CLOSING) {
		alert("Websocket is closed");
		ws = new WebSocket(`ws://${host}/ws`);
	}

	pc.onicecandidate = event => {
		console.log(event.candidate.toJSON())
		if (event.candidate) {
			ws.send(JSON.stringify({
				action: "offer",
				candidate: event.candidate.toJSON()
			}))
		}
	}

	const offerDescription = await pc.createOffer()
	await pc.setLocalDescription(offerDescription)

	const offer = {
		sdp: offerDescription.sdp,
		type: offerDescription.type
	}

	console.log(offer)
	ws.send(
		JSON.stringify({
			action: "create",
			data: {
				sdp: offer,
			},
		})
	);
});

$(".joinBtn").on("click", () => {
	id = $("#game_id_input").val();
	joinGame(id);
});

$(".leaveBtn").on("click", () => {
	console.log("let me out");
	sessionStorage.clear();
	window.location.reload();
});


