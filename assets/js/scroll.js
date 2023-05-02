var width = document.getElementById('slider').scrollWidth;
function sleep(time) {
    return new Promise((resolve) => setTimeout(resolve, time));
}
var progress = 0
var flag = true
document.addEventListener("touchstart", function (e) {
    var element = document.getElementById('slider');

    if (e.target == element) {
        flag = false
    }
    return true
}, {passive: true});
const sb = new ScrollBooster({
    viewport: document.querySelector('#slider'),
    scrollMode: 'transform',
    direction: 'horizontal',
    emulateScroll: true,
    lockScrollOnDragDirection: 'all',
    onClick: (s, ev, mobile) => {
        const isLink = ev.target.nodeName.toLowerCase() === 'link';
        if (isLink) {
            ev.target.click()
        }
        flag = false
    },
    onUpdate: (s) => {
        if (s.borderCollision.right) {
            progress = 0
        }
    },
    onPointerDown: (s) => {
        flag = false
    }

});
setInterval(function () {
    if (!flag) {
        return
    }
    sb.scrollTo({ x: progress * width, y: 0 })
    progress += 0.125
}, 3000)