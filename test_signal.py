import requests
import json
import time

SIGNAL_URL = "https://signal.zgentime.com"
ROOM_ID = "test_room_123"
SENDER_DATA = "192.0.2.1:12345"
RECEIVER_DATA = "192.0.2.2:54321"

print(f"Testing Signal Server: {SIGNAL_URL}")

# --- Test POST (Sender sends data) ---
print("\n--- POST: Sender sends data ---")
post_data_sender = {
    "room": ROOM_ID,
    "role": "sender",
    "data": SENDER_DATA
}
try:
    response = requests.post(SIGNAL_URL, json=post_data_sender)
    response.raise_for_status() # Raise an exception for HTTP errors
    print(f"Sender POST Response Status: {response.status_code}")
    print(f"Sender POST Response Body: {response.json()}")
except requests.exceptions.RequestException as e:
    print(f"Error during Sender POST: {e}")
    if hasattr(e, 'response') and e.response is not None:
        print(f"Response Status: {e.response.status_code}")
        print(f"Response Body: {e.response.text}")

# --- Test POST (Receiver sends data) ---
print("\n--- POST: Receiver sends data ---")
post_data_receiver = {
    "room": ROOM_ID,
    "role": "receiver",
    "data": RECEIVER_DATA
}
try:
    response = requests.post(SIGNAL_URL, json=post_data_receiver)
    response.raise_for_status() # Raise an exception for HTTP errors
    print(f"Receiver POST Response Status: {response.status_code}")
    print(f"Receiver POST Response Body: {response.json()}")
except requests.exceptions.RequestException as e:
    print(f"Error during Receiver POST: {e}")
    if hasattr(e, 'response') and e.response is not None:
        print(f"Response Status: {e.response.status_code}")
        print(f"Response Body: {e.response.text}")

# --- Test GET (Sender requests Receiver's data) ---
print("\n--- GET: Sender requests Receiver's data ---")
get_url_sender = f"{SIGNAL_URL}?room={ROOM_ID}&role=sender"
try:
    response = requests.get(get_url_sender)
    response.raise_for_status()
    print(f"Sender GET Response Status: {response.status_code}")
    print(f"Sender GET Response Body: {response.text}")
    if response.text == RECEIVER_DATA:
        print("Sender GET: Successfully retrieved Receiver's data.")
    else:
        print("Sender GET: Data mismatch or not found.")
except requests.exceptions.RequestException as e:
    print(f"Error during Sender GET: {e}")
    if hasattr(e, 'response') and e.response is not None:
        print(f"Response Status: {e.response.status_code}")
        print(f"Response Body: {e.response.text}")

# --- Test GET (Receiver requests Sender's data) ---
print("\n--- GET: Receiver requests Sender's data ---")
get_url_receiver = f"{SIGNAL_URL}?room={ROOM_ID}&role=receiver"
try:
    response = requests.get(get_url_receiver)
    response.raise_for_status()
    print(f"Receiver GET Response Status: {response.status_code}")
    print(f"Receiver GET Response Body: {response.text}")
    if response.text == SENDER_DATA:
        print("Receiver GET: Successfully retrieved Sender's data.")
    else:
        print("Receiver GET: Data mismatch or not found.")
except requests.exceptions.RequestException as e:
    print(f"Error during Receiver GET: {e}")
    if hasattr(e, 'response') and e.response is not None:
        print(f"Response Status: {e.response.status_code}")
        print(f"Response Body: {e.response.text}")

print("\nSignal server test complete.")
