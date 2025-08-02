<?php
ini_set('display_errors', 1);
error_reporting(E_ALL);

// Simple in-memory store using a file.
// For a real application, use Redis, Memcached, or a database.
$storageFile = '/tmp/stun_forward_session.json';

function get_store() {
    global $storageFile;
    if (!file_exists($storageFile)) {
        return [];
    }
    $data = file_get_contents($storageFile);
    return json_decode($data, true) ?: [];
}

function save_store($store) {
    global $storageFile;
    file_put_contents($storageFile, json_encode($store));
}

header("Content-Type: application/json");
header("Access-Control-Allow-Origin: *");
header("Access-Control-Allow-Methods: GET, POST, OPTIONS");
header("Access-Control-Allow-Headers: Content-Type");

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $raw = file_get_contents("php://input");
    $data = json_decode($raw, true);

    if (!$data || !isset($data['room']) || !isset($data['role']) || !isset($data['data'])) {
        http_response_code(400);
        echo json_encode(["error" => "Missing or invalid room/role/data"]);
        exit;
    }

    $store = get_store();
    if (!isset($store[$data['room']])) {
        $store[$data['room']] = [];
    }
    $store[$data['room']][$data['role']] = $data['data'];
    save_store($store);

    echo json_encode(["status" => "ok"]);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] === 'GET') {
    $room = $_GET['room'] ?? null;
    $role = $_GET['role'] ?? null;

    if (!$room || !$role || !in_array($role, ['client', 'server'])) {
        http_response_code(400);
        echo json_encode(["error" => "Missing or invalid room/role"]);
        exit;
    }

    $store = get_store();
    // Get data for the requested role (not peer role)
    $data = $store[$room][$role] ?? null;

    if ($data) {
        echo $data;
    } else {
        http_response_code(404);
        echo json_encode(["error" => "Peer not found"]);
    }
    exit;
}

http_response_code(405);
echo json_encode(["error" => "Unsupported method"]);