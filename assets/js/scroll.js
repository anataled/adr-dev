import Swiper from 'https://cdn.jsdelivr.net/npm/swiper@10/swiper-bundle.min.mjs'

const swiper = new Swiper('.swiper', {
    loop: true,
    slidesPerView: 3,
    speed: 600,
    autoplay: {
        delay: 500,
    },
    breakpoints: {
        640: { // sm
            slidesPerView: 4
        },
        768: { // md
            slidesPerView: 4
        },
        1024: { // lg
            slidesPerView: 4
        },
        1280: { // xl
            slidesPerView: 6
        },
        1536: { // 2xl
            slidesPerView: 6
        },
    }
});
