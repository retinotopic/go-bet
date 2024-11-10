<script lang="ts">
    import Card from './card.svelte';
    
    let { cards }: { cards: string[] } = $props();
    const cardOffset = 90; 
      
    let totalWidth = cardOffset * (cards.length - 1) ;
  </script>
  
  <div class="hand-container" style:width="{totalWidth}px">
    {#each cards as _,i}
      <div class="card-wrapper" style:left="{i * cardOffset}px">
        <Card ranksuit={cards[i]}/>
      </div>
    {/each}
  </div>
  
  <style>
    .hand-container {
      position: relative;
      margin: 20px ;
    }
  
    .card-wrapper {
      position: absolute;
      top: 0;
      transition: all 0.3s ease;
    }
  
    .card-wrapper:hover {
      z-index: 10;
    }
  </style>