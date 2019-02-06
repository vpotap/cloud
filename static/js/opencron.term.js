;function OpencronTerm() {
    this.socket = null;
    this.term = null;
    this.args = arguments;
    // console.log(this.args);
    this.contextPath = (window.location.protocol === "https:" ? "wss://" : "ws://") + document.domain;
    // console.log(window.location.host)
    this.backgroundColor = '#000000';
    this.fontColor = "#cccccc";
    this.termContainer = $("#terminal-container");
    this.open();
}

;OpencronTerm.prototype.getSize = function () {
    var cols = Math.floor($(window).innerWidth() / 7.2261);//基于fontSize=12的cols参考值
    var span = $("<span>");
    this.termContainer.append(span);
    var array = ['q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', 'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'z', 'x', 'c', 'v', 'b', 'n', 'm', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0'];
    var i = 0;
    while (true) {
        span.text(span.text() + (i < array.length ? array[i] : array[i % (array.length - 1)]));
        if ($(window).width() < span.width()) {
            cols = i - 1;
            break;
        }
        ++i;
    }
    span.remove();
    return {
        width: $(window).innerWidth(),
        height: $(window).innerHeight() - $("#appbar").outerHeight(),
        cols: cols,
        rows: Math.floor(($(window).innerHeight() - $("#appbar").outerHeight() - 5) / 17.15)
    };
}

;OpencronTerm.prototype.open = function () {

    var self = this;
    self.offset = self.getSize();

    self.term = new Terminal({
        termName: "xterm",
        useStyle: true,
        screenKeys: true,
        cursorBlink: false,
        convertEol: true,
        scrollback: 1000,
        tabstopwidth: 4,
        colors: Terminal.xtermColors
    });
    self.term.open(self.termContainer.empty()[0]);
    self.theme(self.args[2]);
    self.term.fit();
    self.term.resize(self.offset.cols, self.offset.rows);
    $(window).resize(function () {
        var currSize = self.getSize();
        if (self.offset.cols != currSize.cols || self.offset.rows != currSize.rows) {
            self.offset = currSize;
            $(".terminal").height(self.offset.height);
            self.term.resize(self.offset.cols, self.offset.rows);
            $.ajax({
                headers: {"csrf": self.args[1]},
                url: '/terminal/resize.do',
                type:'POST',
                cache:false,
                data: {
                    "token": self.args[0],
                    "id":self.args[4],
                    "cols": self.offset.cols,
                    "rows": self.offset.rows,
                    "width": self.offset.width,
                    "heigh": self.offset.heigh
                }
            });
        }
    });

    var params = '?' + $.param(self.offset || {});
    params += "&pod="+this.args[4] +"&container="+this.args[3]+"&namespace="+this.args[1]+"&username="+this.args[2]+"&token="+this.args[0]+"&timestamp="+this.args[5]+"&cluster="+this.args[6];
    // console.log(this.args)
    // console.log(params);

    window.WebSocket = window.WebSocket || window.MozWebSocket;
    // console.log(this.contextPath)
    if(window.WebSocket) {
        self.socket = new WebSocket('ws:'+document.domain+':8999/tty' + params);
        // self.socket = new WebSocket(this.contextPath + '/terminal.ws' + params);
    }else {
        self.socket = SockJS("http://" + window.location.host + "/terminal.js" + params);
    }


    self.socket.onopen = function () {
        self.term.attach(self.socket);
        self._initialized = true;
    };

    self.socket.onerror = function () {
        self.term.write("Sorry! zcloud terminal connect error!please try again.\n");
        window.clearInterval(self.term._blink);
    };

    self.socket.onclose = function () {
        self.term.write("Thank you for using zcloud terminal! bye...");
        //清除光标闪烁
        window.clearInterval(self.term._blink);
        self.termClosed = true;
        document.title = "Terminal Disconnect";
        $('<div class="modal-backdrop in" id="backdrop">').appendTo('body');
        //转移焦点到零时的输入框,主要是为了接管term对键盘的监听(终端已经logout的情况下,再点击Enter则关闭当前页面)
        $("<input type='text' id='unfocusinput' width='0px' height='0px' style='border:0;outline:none;position: absolute;top: -1000px;left: -1000px;'>").appendTo('body');
        document.getElementById("unfocusinput").focus();
        $(".terminal-cursor").remove();
    };


    $("#sendInput").focus(function () {
        self.inFocus = true;
    }).blur(function () {
        self.inFocus = false;
    });
    var sockets = this.socket;
    // 30分钟自动关闭
    setTimeout(function () {
        sockets.send("exit\r\n")
    },1800000)
    $(document).keypress(function (e) {
        var keyCode = (e.keyCode) ? e.keyCode : e.charCode;
        if (keyCode === 13) {
            //在中文输入框里点击Enter按钮触发发送事件.
            if (self.hasOwnProperty("inFocus") && self.inFocus) {
                $("#sendBtn").click();
            }
            //(终端已经logout的情况下,再点击Enter则关闭当前页面
            if (self.hasOwnProperty("termClosed")) {
                self.term.close();
                window.close();
            }
        }
    });
};


;OpencronTerm.prototype.theme = function () {
    'use strict';
    if (this.themeName == arguments[0]) {
        return;
    }
    this.themeName = arguments[0] || "default";
    switch (this.themeName) {
        case "green":
            this.backgroundColor = '#000000';
            this.fontColor = "#00FF00";
            break;
        case "black":
            this.backgroundColor = '#000000';
            this.fontColor = "#FFFFFF";
            break;
        case "gray":
            this.backgroundColor = '#000000';
            this.fontColor = "#AAAAAA";
            break;
        default:
            this.backgroundColor = '#000000';
            this.fontColor = "#cccccc";
            break;
    }
    /**
     * 别动,很神奇....非读熟源码是写不出下面的代码的.
     */
    $("#term-style").remove();
    var style = document.getElementById('term-style');
    var head = document.getElementsByTagName('head')[0];
    var style = document.createElement('style');
    style.id = 'term-style';
    // textContent doesn't work well with IE for <style> elements.
    style.innerHTML = ''
        + '.terminal:not(.xterm-cursor-style-underline):not(.xterm-cursor-style-bar) .terminal-cursor {\n'
        + '  color: ' + this.backgroundColor + ';\n'
        + '  background: ' + this.fontColor + ';\n'
        + '}\n'
        + '.terminal:not(.focus) .terminal-cursor {\n'
        + '  outline: 1px solid ' + this.backgroundColor + ';\n'
        + '  outline-offset: -1px;\n'
        + '  background-color: transparent;\n'
        + '}\n';
    head.insertBefore(style, head.lastChild);

    $('body').css("background-color", this.backgroundColor);

    $(".terminal").css({
        "background-color": this.backgroundColor,
        "color": this.fontColor
    }).focus();

    // //同步到后台服务器
    // $.ajax({
    //     headers:{"csrf":this.args[1]},
    //     url:"/terminal/theme.do",
    //     type:"POST",
    //     cache:false,
    //     data:{
    //         "token":this.args[0],
    //         "id":this.args[3],
    //         "theme": this.themeName
    //     }
    // });
};