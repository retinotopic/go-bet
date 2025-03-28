<script lang="ts">

    let { deadline }: { deadline: number } = $props();
    var timerId :number;
    var started :boolean = false;
    let timeLeft = $state(0);
    $effect(() => {
                SetTimer(deadline)
    });
    function SetTimer(dline: number) {
            if (started === false) {
                if (dline > 0) {
                        timerId = setInterval(updateTime, 100,dline);
                } else {
                        timeLeft = 0;
                } 
                started = true
            }
    }
    function updateTime(dline: number) {
        const now = Math.floor(Date.now() / 1000); 
        const diff: number = dline - now;
        timeLeft = diff > 0 ? diff : 0;
        if (diff <= 0) {
            clearInterval(timerId)
            started = false
        }
    }
    </script>
    
    {#if timeLeft > 0}
        <input type="range" bind:value={timeLeft}/>
    {/if}
