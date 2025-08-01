<?php
// signal.php

session_start();
header('Content-Type: application/json');

// In-memory store, only survives per-process (good for CLI debug or FPM long-run script)
// Real-world: Replace with APCu, Redis, SQLite, or file cache

if (!isset($_SESSION['signal_rooms'])) {
    $_SESSION['signal_rooms'] = [];
}

$method = $_SERVER['REQUEST_METHOD'];

function respond($status, $message = '') {
    http_response_code($status);
    if ($message !== '') echo json_encode(["error" => $message]);
    exit;
}

function get_room_ref(&$store, $room) {
    if (!isset($store[$room])) {
        $store[$room] = ["sender" => null, "receiver" => null, "timestamp" => time()];
    }
    return $store[$room];
}

$room = $_GET['room'] ?? ($_POST['room'] ?? '');
$role = $_GET['role'] ?? ($_POST['role'] ?? '');

if (!$room || !$role || !in_array($role, ['sender', 'receiver'])) {
    respond(400, "Missing or invalid room/role");
}

if ($method === 'POST') {
    $input = json_decode(file_get_contents('php://input'), true);
    if (!isset($input['data'])) respond(400, "Missing data");

    $_SESSION['signal_rooms'][$room][$role] = [
        "data" => $input['data'],
        "timestamp" => time()
    ];
    respond(200);

} elseif ($method === 'GET') {
    $peerRole = $role === 'sender' ? 'receiver' : 'sender';
    $entry = $_SESSION['signal_rooms'][$room][$peerRole] ?? null;

    if ($entry && time() - $entry['timestamp'] < 60) {
        echo $entry['data'];
        exit;
    } else {
        respond(204); // No content
    }

} else {
    respond(405, "Only GET/POST allowed");
}
