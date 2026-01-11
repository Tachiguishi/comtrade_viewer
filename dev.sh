#!/bin/bash
# Start both backend and frontend in development mode

echo "Starting ComTrade Viewer development servers..."

# Start backend in background
cd backend
echo "Starting Go backend on :8080..."
go run main.go &
BACKEND_PID=$!

# Wait a bit for backend to start
sleep 2

# Start frontend
cd ../frontend
echo "Starting Vue frontend on :5173..."
npm run dev &
FRONTEND_PID=$!

# Trap to clean up on exit
trap "echo 'Stopping servers...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit" INT TERM

echo ""
echo "✓ Backend running at http://localhost:8080"
echo "✓ Frontend running at http://localhost:5173"
echo ""
echo "Press Ctrl+C to stop both servers"

# Wait for both processes
wait
