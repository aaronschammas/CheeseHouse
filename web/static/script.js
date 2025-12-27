// Juego de timing para CheeseHouse

document.addEventListener('DOMContentLoaded', function() {
    const startBtn = document.getElementById('start-btn');
    const timerDisplay = document.getElementById('timer');
    const targetTimeDisplay = document.getElementById('target-time');
    const resultDiv = document.getElementById('result');
    const resultTitle = document.getElementById('result-title');
    const resultMessage = document.getElementById('result-message');
    const voucherInfo = document.getElementById('voucher-info');
    const voucherCode = document.getElementById('voucher-code');
    const discount = document.getElementById('discount');
    const expiryDate = document.getElementById('expiry-date');

    let startTime;
    let timerInterval;
    let targetTime = 0;
    let gameStarted = false;
    let gameFinished = false;

    // Obtener tiempo objetivo al cargar la página
    fetchTargetTime();

    // Event listeners
    startBtn.addEventListener('click', startGame);
    document.addEventListener('keydown', handleKeyPress);

    async function fetchTargetTime() {
        try {
            const response = await fetch('/api/game/target');
            const data = await response.json();
            if (data.success) {
                targetTime = data.target_time;
                targetTimeDisplay.textContent = targetTime.toFixed(1);
            } else {
                showError('Error al obtener tiempo objetivo');
            }
        } catch (error) {
            console.error('Error fetching target time:', error);
            showError('Error de conexión');
        }
    }

    function startGame() {
        if (gameStarted) return;

        gameStarted = true;
        gameFinished = false;
        startTime = Date.now();

        startBtn.textContent = '¡Presiona ESPACIO ahora!';
        startBtn.disabled = true;
        startBtn.className = 'btn btn-danger';

        // Iniciar cronómetro
        timerInterval = setInterval(updateTimer, 10);

        // Auto-terminar después de 30 segundos como máximo
        setTimeout(() => {
            if (!gameFinished) {
                finishGame();
            }
        }, 30000);
    }

    function updateTimer() {
        const elapsed = (Date.now() - startTime) / 1000;
        timerDisplay.textContent = elapsed.toFixed(2);
    }

    function handleKeyPress(event) {
        if (event.code === 'Space' && gameStarted && !gameFinished) {
            event.preventDefault();
            finishGame();
        }
    }

    function finishGame() {
        if (gameFinished) return;

        gameFinished = true;
        clearInterval(timerInterval);

        const endTime = Date.now();
        const reactionTime = (endTime - startTime) / 1000;

        startBtn.textContent = 'Juego Terminado';
        startBtn.disabled = true;

        submitResult(reactionTime);
    }

    async function submitResult(reactionTime) {
        try {
            const response = await fetch('/api/game/submit', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    reaction_time: reactionTime,
                    target_time: targetTime,
                    timestamp: new Date().toISOString()
                })
            });

            const data = await response.json();

            if (data.success) {
                showSuccess(data);
            } else {
                showError(data.message || 'Error desconocido');
            }
        } catch (error) {
            console.error('Error submitting result:', error);
            showError('Error de conexión al enviar resultado');
        }
    }

    function showSuccess(data) {
        resultDiv.style.display = 'block';
        resultDiv.className = 'result success';
        resultTitle.textContent = '¡Felicitaciones!';
        resultMessage.textContent = data.message;

        if (data.voucher_code) {
            voucherInfo.style.display = 'block';
            voucherCode.textContent = data.voucher_code;
            discount.textContent = data.discount;
            expiryDate.textContent = data.expiry_date;
        }

        // Reset game after 10 seconds
        setTimeout(resetGame, 10000);
    }

    function showError(message) {
        resultDiv.style.display = 'block';
        resultDiv.className = 'result error';
        resultTitle.textContent = '¡Ups!';
        resultMessage.textContent = message;
        voucherInfo.style.display = 'none';

        // Reset game after 5 seconds
        setTimeout(resetGame, 5000);
    }

    function resetGame() {
        gameStarted = false;
        gameFinished = false;
        timerDisplay.textContent = '0.00';
        startBtn.textContent = 'Iniciar Juego';
        startBtn.disabled = false;
        startBtn.className = 'btn btn-primary';
        resultDiv.style.display = 'none';

        // Obtener nuevo tiempo objetivo
        fetchTargetTime();
    }

    // Prevenir scroll con barra espaciadora
    document.addEventListener('keydown', function(event) {
        if (event.code === 'Space') {
            event.preventDefault();
        }
    });
});