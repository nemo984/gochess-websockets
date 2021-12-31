var board = null;
var game = new Chess();
var $status = $("#status");
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
            .then(function () {})
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

    $status.html(status);
    $fen.html(game.fen());
    $pgn.html(game.pgn());
}

function joinGame(id) {
    if (ws.readyState === WebSocket.CLOSED) {
        alert("Websocket is closed");
    }
    if (id === "") {
        alert("GAME ID INPUT IS EMPTY");
    }
    ws.send(
        JSON.stringify({
            action: "join",
            data: { id: id },
        })
    );
}

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

ws.onopen = (e) => {
    const gameID = sessionStorage.getItem("id");
    if (gameID !== null) {
        joinGame(gameID);
    }
};

ws.onmessage = (e) => {
    json = JSON.parse(e.data);
    console.log(json);

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
                return;
            case "Game joined as black":
                myColor = "b";
                board.flip();
                return;
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

$(".createBtn").on("click", () => {
    if (ws.readyState === WebSocket.CLOSED) {
        alert("Websocket is closed");
    }

    ws.send(
        JSON.stringify({
            action: "create",
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
