<script lang="ts">
    let card: HTMLElement;

    let { ranksuit }: { ranksuit: string } = $props();
    let rotateX =$state(0);
    let rotateY =$state(0);
    let scale = $state(1);
    let rotateXs ="";
    let rotateYs = "";
    function handleMouseMove(event: MouseEvent) {
  
      const rect = card.getBoundingClientRect();
      const x = (event.clientX - rect.left) / rect.width;
      const y = (event.clientY - rect.top) / rect.height;
      
      rotateYs = ((x - 0.5) * 40).toFixed(2);
      rotateXs = (((y - 0.5) * -40)).toFixed(2);
      rotateY = parseInt(rotateYs, 10)
      rotateX = parseInt(rotateXs, 10)
          scale = 1.1;
    }
    
    function handleMouseLeave() {
      rotateX = 0;
      rotateY = 0;
      scale = 1
    }
  </script>
  
  <div class="card-container">
    <div 
      class="card"
      role="list"
      bind:this={card}
      onmousemove={handleMouseMove}
      onmouseleave={handleMouseLeave}
      style:transform="rotateX({rotateX}deg) rotateY({rotateY}deg) scale({scale})"
    >
      <div class="card-face front">
              <img src="'/lib/cardsrc/{ranksuit}.png'" alt="card" width="200" height="300">
        </div>
          
    </div>
  </div>
  
  <style>
    .card-container {
      perspective: 1000px;
      width: 200px;
      height: 300px;
    }
  
    .card {
      position: relative;
      width: 100%;
      height: 100%;
      transform-style: preserve-3d;
      transition: transform 0.1s ease-out;
    }
  
    .card-face {
      position: absolute;
      width: 100%;
      height: 100%;
      backface-visibility: hidden;
      border-radius: 10px;
    }
  
    .front {
      background: linear-gradient(145deg, #ffffff, #e6e6e6);
      box-shadow: 
        0 5px 15px rgba(0,0,0,0.3),
        inset 0 0 15px rgba(255,255,255,0.5);
    }
  
    .card:hover .front::before {
      opacity: 0.8;
    }
  </style>