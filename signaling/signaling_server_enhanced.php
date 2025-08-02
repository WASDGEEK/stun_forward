<?php
ini_set('display_errors', 1);
error_reporting(E_ALL);

// Enhanced signaling server with mapping sync and auto-cleanup
$storageFile = '/tmp/stun_forward_enhanced.json';
$ROOM_EXPIRY_MINUTES = 5; // Auto cleanup after 5 minutes of inactivity

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
    file_put_contents($storageFile, json_encode($store, JSON_PRETTY_PRINT));
}

function cleanup_expired_rooms() {
    global $ROOM_EXPIRY_MINUTES;
    $store = get_store();
    $current_time = time();
    $expired_rooms = [];
    
    foreach ($store as $room_id => $room_data) {
        if (isset($room_data['last_activity'])) {
            $inactive_minutes = ($current_time - $room_data['last_activity']) / 60;
            if ($inactive_minutes > $ROOM_EXPIRY_MINUTES) {
                $expired_rooms[] = $room_id;
            }
        }
    }
    
    foreach ($expired_rooms as $room_id) {
        unset($store[$room_id]);
        error_log("Cleaned up expired room: $room_id");
    }
    
    if (!empty($expired_rooms)) {
        save_store($store);
    }
    
    return count($expired_rooms);
}

function touch_room_activity($room_id) {
    $store = get_store();
    if (!isset($store[$room_id])) {
        $store[$room_id] = [
            'created_at' => time(),
            'version' => 1,
            'participants' => []
        ];
    }
    
    $store[$room_id]['last_activity'] = time();
    save_store($store);
    return $store;
}

function update_participant_data($room_id, $role, $data) {
    $store = touch_room_activity($room_id);
    
    // Parse the data to handle mapping updates
    $parsed_data = json_decode($data, true);
    
    if (!isset($store[$room_id]['participants'][$role])) {
        $store[$room_id]['participants'][$role] = [
            'first_seen' => time(),
            'version' => 1,
            'data' => $data
        ];
    } else {
        $store[$room_id]['participants'][$role]['version']++;
        $store[$room_id]['participants'][$role]['data'] = $data;
    }
    
    $store[$room_id]['participants'][$role]['last_updated'] = time();
    $store[$room_id]['version']++;
    
    // If this is a client with mapping updates, trigger server notification
    if ($role === 'client' && $parsed_data && isset($parsed_data['mappings'])) {
        $store[$room_id]['mapping_update_pending'] = true;
        $store[$room_id]['mapping_version'] = time();
    }
    
    save_store($store);
    return $store[$room_id];
}

function get_participant_data($room_id, $role) {
    $store = get_store();
    
    if (!isset($store[$room_id]) || !isset($store[$room_id]['participants'][$role])) {
        return null;
    }
    
    // Touch activity when data is accessed
    touch_room_activity($room_id);
    
    return $store[$room_id]['participants'][$role]['data'];
}

function check_mapping_updates($room_id, $last_known_version = 0) {
    $store = get_store();
    
    if (!isset($store[$room_id])) {
        return null;
    }
    
    $room_data = $store[$room_id];
    $current_mapping_version = $room_data['mapping_version'] ?? 0;
    
    if ($current_mapping_version > $last_known_version) {
        return [
            'has_update' => true,
            'version' => $current_mapping_version,
            'client_data' => $room_data['participants']['client']['data'] ?? null
        ];
    }
    
    return ['has_update' => false, 'version' => $current_mapping_version];
}

header("Content-Type: application/json");
header("Access-Control-Allow-Origin: *");
header("Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS");
header("Access-Control-Allow-Headers: Content-Type, X-Mapping-Version");

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

// Cleanup expired rooms on every request
cleanup_expired_rooms();

// POST: Register/Update participant data
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $raw = file_get_contents("php://input");
    $data = json_decode($raw, true);

    if (!$data || !isset($data['room']) || !isset($data['role']) || !isset($data['data'])) {
        http_response_code(400);
        echo json_encode(["error" => "Missing or invalid room/role/data"]);
        exit;
    }

    $room_data = update_participant_data($data['room'], $data['role'], $data['data']);
    
    echo json_encode([
        "status" => "ok",
        "room_version" => $room_data['version'],
        "participant_version" => $room_data['participants'][$data['role']]['version'],
        "mapping_version" => $room_data['mapping_version'] ?? 0
    ]);
    exit;
}

// GET: Retrieve participant data
if ($_SERVER['REQUEST_METHOD'] === 'GET') {
    $room = $_GET['room'] ?? null;
    $role = $_GET['role'] ?? null;
    $check_updates = $_GET['check_updates'] ?? false;
    $last_mapping_version = intval($_GET['last_mapping_version'] ?? 0);

    if (!$room || !$role || !in_array($role, ['client', 'server'])) {
        http_response_code(400);
        echo json_encode(["error" => "Missing or invalid room/role"]);
        exit;
    }

    // Check for mapping updates (for server polling)
    if ($check_updates) {
        $update_info = check_mapping_updates($room, $last_mapping_version);
        echo json_encode($update_info);
        exit;
    }

    $data = get_participant_data($room, $role);

    if ($data) {
        echo $data;
    } else {
        http_response_code(404);
        echo json_encode(["error" => "Participant not found"]);
    }
    exit;
}

// PUT: Update only mappings (for hot updates)
if ($_SERVER['REQUEST_METHOD'] === 'PUT') {
    $raw = file_get_contents("php://input");
    $data = json_decode($raw, true);

    if (!$data || !isset($data['room']) || !isset($data['mappings'])) {
        http_response_code(400);
        echo json_encode(["error" => "Missing room or mappings"]);
        exit;
    }

    $room_id = $data['room'];
    $store = get_store();
    
    if (!isset($store[$room_id]['participants']['client'])) {
        http_response_code(404);
        echo json_encode(["error" => "Client not found in room"]);
        exit;
    }

    // Update client mappings
    $client_data = json_decode($store[$room_id]['participants']['client']['data'], true);
    $client_data['mappings'] = $data['mappings'];
    
    update_participant_data($room_id, 'client', json_encode($client_data));
    
    echo json_encode([
        "status" => "mappings_updated",
        "mapping_version" => $store[$room_id]['mapping_version']
    ]);
    exit;
}

// DELETE: Clean up room manually
if ($_SERVER['REQUEST_METHOD'] === 'DELETE') {
    $room = $_GET['room'] ?? null;
    
    if (!$room) {
        http_response_code(400);
        echo json_encode(["error" => "Missing room parameter"]);
        exit;
    }
    
    $store = get_store();
    if (isset($store[$room])) {
        unset($store[$room]);
        save_store($store);
        echo json_encode(["status" => "room_deleted"]);
    } else {
        http_response_code(404);
        echo json_encode(["error" => "Room not found"]);
    }
    exit;
}

http_response_code(405);
echo json_encode(["error" => "Unsupported method"]);
?>