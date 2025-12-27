// Configuración del juego
const GAME_CONFIG = {
  winDiscount: 50,
  loseDiscount: 10,
  tolerance: 0,
  minTargetTime: 5,
  maxTargetTime: 12,
  animationDuration: 300,
}

// Clase principal del juego
class TimingGame {
  constructor() {
    this.startTime = null
    this.animationFrameId = null
    this.targetTime = 0
    this.isGameRunning = false
    this.hasWon = false

    // Cache de elementos DOM
    this.elements = this.cacheElements()

    // Inicializar el juego
    this.init()
  }

  // Cachear elementos DOM para mejor rendimiento
  cacheElements() {
    return {
      targetTime: document.getElementById("targetTime"),
      timerDisplay: document.getElementById("timerDisplay"),
      startButton: document.getElementById("startButton"),
      stopButton: document.getElementById("stopButton"),
      resultMessage: document.getElementById("resultMessage"),
      modalOverlay: document.getElementById("modalOverlay"),
      modalClose: document.getElementById("modalClose"),
      gameForm: document.getElementById("gameForm"),
      nombreInput: document.getElementById("nombre"),
      apellidoInput: document.getElementById("apellido"),
      telefonoInput: document.getElementById("telefono"),
    }
  }

  // Inicializar el juego
  init() {
    this.generateTargetTime()
    this.bindEvents()
    this.resetForm()

    // Añadir clase para animaciones si está soportado
    if (window.requestAnimationFrame) {
      document.body.classList.add("animations-supported")
    }
  }

  // Vincular eventos
  bindEvents() {
    this.elements.startButton.addEventListener("click", () => this.startGame())
    this.elements.stopButton.addEventListener("click", () => this.stopGame())
    this.elements.gameForm.addEventListener("submit", (e) => this.handleFormSubmit(e))
    this.elements.modalClose.addEventListener("click", () => this.closeModal())
    this.elements.modalOverlay.addEventListener("click", (e) => {
      if (e.target === this.elements.modalOverlay) {
        this.closeModal()
      }
    })

    // Eventos de teclado para accesibilidad
    document.addEventListener("keydown", (e) => this.handleKeyPress(e))

    // Prevenir zoom en iOS al hacer doble tap
    this.elements.startButton.addEventListener("touchend", (e) => {
      e.preventDefault()
      this.startGame()
    })

    this.elements.stopButton.addEventListener("touchend", (e) => {
      e.preventDefault()
      this.stopGame()
    })
  }

  // Manejar teclas para accesibilidad
  handleKeyPress(e) {
    if (e.code === "Space") {
      e.preventDefault()
      if (this.isGameRunning) {
        this.stopGame()
      } else if (!this.elements.startButton.classList.contains("hidden")) {
        this.startGame()
      }
    }
    if (e.code === "Escape" && this.elements.modalOverlay.classList.contains("show")) {
      this.closeModal()
    }
  }

  // Generar tiempo objetivo desde el backend
  async generateTargetTime() {
    try {
      const response = await fetch('/api/game/target');
      const data = await response.json();
      if (data.success) {
        this.targetTime = data.target_time;
        this.elements.targetTime.textContent = this.targetTime;
      } else {
        console.error('Error obteniendo tiempo objetivo:', data);
        // Fallback local
        this.targetTime = 7.2;
        this.elements.targetTime.textContent = this.targetTime;
      }
    } catch (error) {
      console.error('Error de red:', error);
      // Fallback local
      this.targetTime = 7.2;
      this.elements.targetTime.textContent = this.targetTime;
    }
  }

  // Actualizar cronómetro con alta precisión
  updateTimer() {
    const currentTime = (performance.now() - this.startTime) / 1000
    this.elements.timerDisplay.textContent = currentTime.toFixed(2)

    // Añadir efecto visual cuando se acerca al tiempo objetivo
    const difference = Math.abs(currentTime - Number.parseFloat(this.targetTime))
    if (difference < 1 && !this.elements.timerDisplay.classList.contains("pulsing")) {
      this.elements.timerDisplay.classList.add("pulsing")
    }

    if (this.isGameRunning) {
      this.animationFrameId = requestAnimationFrame(() => this.updateTimer())
    }
  }

  // Iniciar el juego
  startGame() {
    if (this.isGameRunning) return

    this.isGameRunning = true
    this.startTime = performance.now()

    // Actualizar UI
    this.toggleButtons(false)
    this.hideElements([this.elements.resultMessage])
    this.closeModal()
    this.elements.timerDisplay.classList.remove("pulsing")

    // Iniciar cronómetro
    this.animationFrameId = requestAnimationFrame(() => this.updateTimer())

    // Feedback háptico en dispositivos móviles
    this.vibrate(50)
  }

  // Detener el juego
  stopGame() {
    if (!this.isGameRunning) return

    this.isGameRunning = false
    cancelAnimationFrame(this.animationFrameId)

    const finalTime = (performance.now() - this.startTime) / 1000
    this.elements.timerDisplay.textContent = finalTime.toFixed(2)
    this.elements.timerDisplay.classList.remove("pulsing")

    const difference = Math.abs(finalTime - Number.parseFloat(this.targetTime))

    // Actualizar UI
    this.toggleButtons(true)

    // Mostrar resultado
    this.showResult(finalTime, difference)

    // Feedback háptico
    this.vibrate(difference <= GAME_CONFIG.tolerance ? [100, 50, 100] : 200)

    // Generar nuevo tiempo objetivo después de un delay
    setTimeout(() => this.generateTargetTime(), 4000)
  }

  // Mostrar resultado del juego
  showResult(finalTime, difference) {
    const resultDiv = this.elements.resultMessage
    const isWin = difference <= GAME_CONFIG.tolerance
    this.hasWon = isWin

    const discount = isWin ? GAME_CONFIG.winDiscount : GAME_CONFIG.loseDiscount
    const message = isWin ? "¡GANASTE!" : "¡PERDISTE!"

    resultDiv.className = `result-message ${isWin ? "win-message" : "lose-message"}`
    resultDiv.innerHTML = `
            ${emoji} <strong>${message}</strong> ${emoji}<br>
            Tu tiempo: <strong>${finalTime.toFixed(2)}s</strong><br>
            Objetivo: <strong>${this.targetTime}s</strong><br>
            <strong>¡Descuento del ${discount}%!</strong>
        `

    this.showElement(resultDiv)

    setTimeout(() => {
      this.showModal()
    }, 1500)
  }

  showModal() {
    this.elements.modalOverlay.classList.remove("hidden")
    // Force reflow for animation
    this.elements.modalOverlay.offsetHeight
    this.elements.modalOverlay.classList.add("show")

    // Focus first input for better UX
    setTimeout(() => {
      this.elements.nombreInput.focus()
    }, 300)

    // Prevent body scroll when modal is open
    document.body.style.overflow = "hidden"
  }

  closeModal() {
    this.elements.modalOverlay.classList.remove("show")

    setTimeout(() => {
      this.elements.modalOverlay.classList.add("hidden")
      document.body.style.overflow = ""
    }, 300)
  }

  // Manejar envío del formulario
  async handleFormSubmit(e) {
    e.preventDefault()

    // Validar datos
    const customerData = this.getCustomerData()
    if (!this.validateCustomerData(customerData)) {
      return
    }

    const gameResult = this.getGameResult()

    // Deshabilitar botón de envío
    const submitButton = e.target.querySelector('button[type="submit"]')
    const originalText = submitButton.innerHTML
    submitButton.disabled = true
    submitButton.innerHTML = '<span class="button-icon">⏳</span> Enviando...'

    try {
      // Simular envío al backend
      await this.submitData(customerData, gameResult)

      // Mostrar mensaje de éxito
      this.showSuccessMessage()

      // Feedback háptico
      this.vibrate([100, 50, 100, 50, 100])
    } catch (error) {
      console.error("Error al enviar datos:", error)
      this.showErrorMessage()
    } finally {
      // Restaurar botón
      submitButton.disabled = false
      submitButton.innerHTML = originalText

      // Reset UI después de un delay
      setTimeout(() => this.resetGameUI(), 3000)
    }
  }

  // Obtener datos del cliente
  getCustomerData() {
    return {
      nombre: this.elements.nombreInput.value.trim(),
      apellido: this.elements.apellidoInput.value.trim(),
      telefono: this.elements.telefonoInput.value.trim(),
    }
  }

  // Validar datos del cliente
  validateCustomerData(data) {
    if (!data.nombre || data.nombre.length < 2) {
      this.showValidationError("Por favor ingresa un nombre válido")
      this.elements.nombreInput.focus()
      return false
    }

    if (!data.apellido || data.apellido.length < 2) {
      this.showValidationError("Por favor ingresa un apellido válido")
      this.elements.apellidoInput.focus()
      return false
    }

    if (!data.telefono || !/^\d{8,15}$/.test(data.telefono.replace(/\s+/g, ""))) {
      this.showValidationError("Por favor ingresa un teléfono válido")
      this.elements.telefonoInput.focus()
      return false
    }

    return true
  }

  // Obtener resultado del juego
  getGameResult() {
    return {
      won: this.hasWon,
      targetTime: Number.parseFloat(this.targetTime),
      finalTime: Number.parseFloat(this.elements.timerDisplay.textContent),
      discount: this.hasWon ? GAME_CONFIG.winDiscount : GAME_CONFIG.loseDiscount,
      timestamp: new Date().toISOString(),
    }
  }

  // Simular envío de datos al backend
  async submitData(customerData, gameResult) {
    try {
      const payload = {
        cliente: {
          nombre: customerData.nombre,
          apellido: customerData.apellido,
          telefono: customerData.telefono
        },
        resultado: {
          gano: gameResult.gano,
          tiempo_objetivo: parseFloat(this.targetTime),
          tiempo_obtenido: gameResult.tiempoObtenido
        }
      };

      const response = await fetch('/api/game/submit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        throw new Error('Error en el servidor');
      }

      const result = await response.json();
      console.log("Respuesta del backend:", result);
      return result;
    } catch (error) {
      console.error('Error enviando datos:', error);
      throw error;
    }
  }

  // Mostrar mensaje de éxito
  showSuccessMessage() {
    const resultDiv = this.elements.resultMessage
    resultDiv.className = "result-message win-message"
    resultDiv.innerHTML = `
            ✅ <strong>¡Datos enviados exitosamente!</strong><br>
            Recibirás tu descuento por WhatsApp pronto.<br>
            <small>¡Gracias por jugar!</small>
        `
  }

  // Mostrar mensaje de error
  showErrorMessage() {
    const resultDiv = this.elements.resultMessage
    resultDiv.className = "result-message lose-message"
    resultDiv.innerHTML = `
            ❌ <strong>Error al enviar datos</strong><br>
            Por favor intenta nuevamente.<br>
            <small>Si el problema persiste, contacta al personal.</small>
        `
  }

  // Mostrar error de validación
  showValidationError(message) {
    // Crear o actualizar mensaje de error temporal
    let errorDiv = document.querySelector(".validation-error")
    if (!errorDiv) {
      errorDiv = document.createElement("div")
      errorDiv.className = "validation-error"
      errorDiv.style.cssText = `
                background: #ffebee;
                color: #c62828;
                padding: 0.5rem;
                border-radius: 8px;
                margin: 0.5rem 0;
                font-size: 0.9rem;
                border: 1px solid #ef5350;
            `
      this.elements.modalOverlay.querySelector(".modal-body").insertBefore(errorDiv, this.elements.gameForm)
    }

    errorDiv.textContent = message

    // Remover después de 3 segundos
    setTimeout(() => {
      if (errorDiv.parentNode) {
        errorDiv.parentNode.removeChild(errorDiv)
      }
    }, 3000)

    // Vibración de error
    this.vibrate(200)
  }

  // Utilidades de UI
  toggleButtons(showStart) {
    if (showStart) {
      this.elements.stopButton.classList.add("hidden")
      this.elements.startButton.classList.remove("hidden")
    } else {
      this.elements.startButton.classList.add("hidden")
      this.elements.stopButton.classList.remove("hidden")
    }
  }

  showElement(element) {
    element.classList.remove("hidden")
  }

  hideElements(elements) {
    elements.forEach((element) => element.classList.add("hidden"))
  }

  // Reset de la UI del juego
  resetGameUI() {
    this.closeModal()
    this.hideElements([this.elements.resultMessage])
    this.resetForm()
    this.generateTargetTime()

    // Remover mensajes de error si existen
    const errorDiv = document.querySelector(".validation-error")
    if (errorDiv && errorDiv.parentNode) {
      errorDiv.parentNode.removeChild(errorDiv)
    }
  }

  // Reset del formulario
  resetForm() {
    this.elements.gameForm.reset()
  }

  // Vibración háptica (si está disponible)
  vibrate(pattern) {
    if ("vibrate" in navigator) {
      navigator.vibrate(pattern)
    }
  }
}

// Inicializar el juego cuando el DOM esté listo
document.addEventListener("DOMContentLoaded", () => {
  // Verificar soporte de características modernas
  if (!window.requestAnimationFrame) {
    console.warn("requestAnimationFrame no soportado, usando setTimeout como fallback")
  }

  // Crear instancia del juego
  window.timingGame = new TimingGame()

  // Registrar service worker si está disponible (para PWA futuro)
  if ("serviceWorker" in navigator) {
    console.log("Service Worker soportado para futuras mejoras PWA")
  }
})

// Manejar visibilidad de la página para pausar el juego si es necesario
document.addEventListener("visibilitychange", () => {
  if (document.hidden && window.timingGame && window.timingGame.isGameRunning) {
    // Opcional: pausar el juego cuando la pestaña no está visible
    console.log("Página oculta - el juego continúa ejecutándose")
  }
})
