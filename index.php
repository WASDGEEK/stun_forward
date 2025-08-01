<?php
ini_set('display_errors', 1);
error_reporting(E_ALL);
session_start();

header("Content-Type: application/json");

// 允许 CORS 测试（可选）
header("Access-Control-Allow-Origin: *");

function json_response($data, $code = 200) {
    http_response_code($code);
    echo json_encode($data);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $raw = file_get_contents("php://input");
    $data = json_decode($raw, true);

    if (!$data || !isset($data['room']) || !isset($data['role']) || !isset($data['data'])) {
        json_response(["error" => "Missing or invalid room/role/data"], 400);
    }

    $_SESSION[$data['room']][$data['role']] = $data['data'];
    json_response(["status" => "ok"]);
}

if ($_SERVER['REQUEST_METHOD'] === 'GET') {
    $room = $_GET['room'] ?? null;
    $role = $_GET['role'] ?? null;

    if (!$room || !$role || !in_array($role, ['sender', 'receiver'])) {
        json_response(["error" => "Missing or invalid room/role"], 400);
    }

    $peer = $role === 'sender' ? 'receiver' : 'sender';
    $data = $_SESSION[$room][$peer] ?? "";
    echo $data;
    exit;
}

json_response(["error" => "Unsupported method"], 405);
