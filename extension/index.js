document.addEventListener("DOMContentLoaded", function () {
  var mySelect = document.getElementById("mySelect");
  
  mySelect.addEventListener("change", function () {
    var selectedValue = this.value;
    // alert(selectedValue);
    var url = ""; // 设置基础 URL

    // 根据选择的值构建完整的 URL
    switch (selectedValue) {
      case "http://127.0.0.1:8080":
        url = "http://127.0.0.1:8080/";
        break;
      case "http://127.0.0.1:8081":
        url = "http://127.0.0.1:8081/";
        break;
      case "http://127.0.0.1:8082":
        url = "http://127.0.0.1:8082/";
        break;
      // 添加更多的选项和相应的 URL
    }

    // 如果 URL 不为空，则进行 AJAX 请求
    if (url !== "") {
      // 发起连接测试请求
      var testConnectionRequest = new XMLHttpRequest();
      testConnectionRequest.timeout = 5000; // 设置超时时间为5秒
      testConnectionRequest.open("GET", url, true);
      testConnectionRequest.onreadystatechange = function () {
        if (testConnectionRequest.readyState === 4) {
          if (testConnectionRequest.status === 200) {
            // 连接成功，弹出成功提示框
            alert("连接:" + url + "成功");
          } else {
            // 连接失败，弹出错误提示框
            alert("连接失败，请检查服务器配置！");
          }
        }
      };
      testConnectionRequest.ontimeout = function () {
        // 请求超时，弹出错误提示框
        alert("连接超时，请检查网络连接或服务器配置！");
      };
      testConnectionRequest.send();

      var loadPrivateKeyButton = document.getElementById("load_privateKey");
      loadPrivateKeyButton.removeEventListener("click", loadPrivateKeyHandler);
      loadPrivateKeyButton.addEventListener("click", loadPrivateKeyHandler);

      var loadRandomButton = document.getElementById("loadRandom");
      loadRandomButton.removeEventListener("click", loadRandomHandler);
      loadRandomButton.addEventListener("click", loadRandomHandler);

      // var GetBalanceButton = document.getElementById("getBalance");
      // GetBalanceButton.removeEventListener("click", GetBalanceHandler);
      // GetBalanceButton.addEventListener("click", GetBalanceHandler);

      
      var GetBalanceButton = document.getElementById("getBalance");
      var GetBalanceButtonAdded = GetBalanceButton.dataset.eventAdded; // 检查是否已添加事件处理程序

      if (!GetBalanceButtonAdded) {
        GetBalanceButton.addEventListener("click", GetBalanceHandler);
        GetBalanceButton.dataset.eventAdded = true; // 设置数据属性表示已添加事件处理程序
      }

      // var transactionButton = document.getElementById("transactionButton");
      // transactionButton.removeEventListener("click", transactionHandler);
      // transactionButton.addEventListener("click", transactionHandler);

      var transactionButton = document.getElementById("transactionButton");
      var transactionButtonAdded = transactionButton.dataset.eventAdded; // 检查是否已添加事件处理程序

      if (!transactionButtonAdded) {
        transactionButton.addEventListener("click", transactionHandler);
        transactionButton.dataset.eventAdded = true; // 设置数据属性表示已添加事件处理程序
      }


      var buttonSubmit = document.getElementById("buttonSubmit");
      var buttonSubmitAdded = buttonSubmit.dataset.eventAdded; // 检查是否已添加事件处理程序

      if (!buttonSubmitAdded) {
        buttonSubmit.addEventListener("click", buttonSubmitHandler);
        buttonSubmit.dataset.eventAdded = true; // 设置数据属性表示已添加事件处理程序
      }

      function loadPrivateKeyHandler() {
        var privateKey = document.getElementById("inputPrivateKey").value;
        console.log("用私钥加载公钥和地址" + privateKey);
        var request = new XMLHttpRequest();
        request.open("POST", url + "walletByPrivatekey", true);
        request.setRequestHeader(
          "Content-Type",
          "application/x-www-form-urlencoded"
        );
        request.onreadystatechange = function () {
          if (request.readyState === 4 && request.status === 200) {
            var response = JSON.parse(request.responseText);
            document.getElementById("inputPublic").value = response.public_key;
            document.getElementById("inputAddress").value =
              response.blockchain_address;
            console.log(response);
          } else {
            console.error("Error: " + request.status);
          }
        };
        var postData = "privatekey=" + encodeURIComponent(privateKey);
        request.send(postData);
        // request.send(privateKey);
      }

      function loadRandomHandler() {
        console.log("随机生成用户钱包：");
        var request = new XMLHttpRequest();
        request.open("POST", url + "wallet", true);
        request.onreadystatechange = function () {
          if (request.readyState === 4 && request.status === 200) {
            var response = JSON.parse(request.responseText);
            document.getElementById("inputPublic").value = response.public_key;
            document.getElementById("inputPrivateKey").value =
              response.private_key;
            document.getElementById("inputAddress").value =
              response.blockchain_address;
            console.info(response);
          } else {
            console.error("Error: " + request.status);
          }
        };
        request.send();
      }

      function GetBalanceHandler() {
        var blockchain_address = document.getElementById("inputAddress").value;


        // 检查区块链地址是否为空
        if (blockchain_address === "") {
          alert("请先加载区块链地址！");
          return;
        }
        var postData = {
          blockchain_address: blockchain_address,
          // 添加其他参数...
        };

        console.log("blockaddress:", JSON.stringify(postData));

        var xhr = new XMLHttpRequest();
        xhr.open("POST", url + "wallet/amount", true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.onreadystatechange = function () {
          if (xhr.readyState === 4) {
            if (xhr.status === 200) {
              var response = JSON.parse(xhr.responseText);
              document.getElementById("Balance").value = response.amount;
              alert("查询成功！" + response.message);
              console.log(response);
            } else {
              console.error("Error: " + xhr.status);
            }
          }
        };
        xhr.send(JSON.stringify(postData));
      }

      function buttonSubmitHandler() {
        var confirm_text = "确定要发送吗?";
        var inputReceiveAddress = document.getElementById("inputReceiveAddress").value;
        var inputAmount = document.getElementById("inputAmount").value;
        var confirm_result = confirm(confirm_text);
        if (confirm_result !== true) {
          alert("取消");
          return;
        }

      
      if(!inputReceiveAddress ||  !inputAmount){
        alert("请填写完整的信息！");
        return;
      }
        
        var amountElement = document.getElementById("Balance");
        var valueElement = document.getElementById("inputAmount");
        var amount = parseFloat(amountElement.value);
        var value = parseFloat(valueElement.value);
        
        if (isNaN(amount) || isNaN(value)) {
          alert("请输入有效的金额");
          return;
        }
        
        // if (amount < value) {
        //   alert("余额不足");
        //   return;
        // }
      
        var transaction_data = {
          sender_private_key: document.getElementById("inputPrivateKey").value,
          sender_blockchain_address: document.getElementById("inputAddress").value,
          sender_public_key: document.getElementById("inputPublic").value,
          recipient_blockchain_address: document.getElementById("inputReceiveAddress").value,
          value: document.getElementById("inputAmount").value,
        };
      
        var xhr = new XMLHttpRequest();
        xhr.open("POST", url + "transaction", true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.onreadystatechange = function () {
          if (xhr.readyState === 4) {
            if (xhr.status === 200) {
              var response = JSON.parse(xhr.responseText);
              console.info(response);
              alert("发送成功");
            } else {
              console.error(xhr.responseText);
              alert("发送失败");
            }
          }
        };
        xhr.send(JSON.stringify(transaction_data));
      }
      
      // // 没验证余额
      // function buttonSubmitHandler() {
      //   var confirm_text = "确定要发送吗?";
      //   var inputReceiveAddress = document.getElementById("inputReceiveAddress").value;
      //   var inputAmount = document.getElementById("inputAmount").value;
      //   var confirm_result = confirm(confirm_text);
      //   if (confirm_result !== true) {
      //     alert("取消");
      //     return;
      //   }

      //   if(!inputReceiveAddress ||  !inputAmount){
      //     alert("请填写完整的信息！");
      //     return;
      //   }

      //   var transaction_data = {
      //     sender_private_key: document.getElementById("inputPrivateKey").value,
      //     sender_blockchain_address:
      //       document.getElementById("inputAddress").value,
      //     sender_public_key: document.getElementById("inputPublic").value,
      //     recipient_blockchain_address: document.getElementById(
      //       "inputReceiveAddress"
      //     ).value,
      //     value: document.getElementById("inputAmount").value,
      //   };

      //   var xhr = new XMLHttpRequest();
      //   xhr.open("POST", url + "transaction", true);
      //   xhr.setRequestHeader("Content-Type", "application/json");
      //   xhr.onreadystatechange = function () {
      //     if (xhr.readyState === 4) {
      //       if (xhr.status === 200) {
      //         var response = JSON.parse(xhr.responseText);
      //         console.info(response);
      //         alert("发送成功");
      //       } else {
      //         console.error(xhr.responseText);
      //         alert("发送失败");
      //       }
      //     }
      //   };
      //   xhr.send(JSON.stringify(transaction_data));
      // }

      function transactionHandler() {
        var blockchain_address = document.getElementById("inputAddress").value;
      
        // 检查区块链地址是否为空
        if (blockchain_address === "") {
          alert("请先加载区块链地址！");
          return;
        }
      
        var postData = {
          blockchain_address: blockchain_address,
          // 添加其他参数...
        };
  
      
        var xhr = new XMLHttpRequest();
        xhr.open("POST", url + "ListTransaction", true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.onreadystatechange = function() {
          if (xhr.readyState === 4) {
            if (xhr.status === 200) {
              var response = JSON.parse(xhr.responseText);
              handleTransactions(response.data);
              alert("查询成功！" + response.message);
              console.log(response);
            } else {
              console.error("Error: " + xhr.status);
            }
          }
        };
        xhr.send(JSON.stringify(postData));
      }
      
      // function handleTransactions(transactions) {
      //   var transactionContainer = document.getElementById("transactionContainer");
      //   transactionContainer.innerHTML = "";
      
      //   var displayBox = document.createElement("pre");
      //   displayBox.style.whiteSpace = "pre-wrap"; // 自动换行
      //   displayBox.textContent = JSON.stringify(transactions, null, 2);
      //   transactionContainer.appendChild(displayBox);
      // }
      
      function handleTransactions(transactions) {
        var transactionContainer = document.getElementById("transactionContainer");
        transactionContainer.innerHTML = "";
      
         // 检查交易数据是否存在
        if (transactions && Array.isArray(transactions)) {
          // 遍历交易数据，创建容器并添加数据
        transactions.forEach(function (transaction) {
          // 创建交易框
          var transactionBox = document.createElement("div");
          transactionBox.className = "transaction-box";
      
          // 创建 from 行
          var fromRow = document.createElement("div");
          fromRow.className = "transaction-row";
          var fromLabel = document.createElement("span");
          fromLabel.className = "label";
          fromLabel.textContent = "From(发送方): ";
          var fromValue = document.createElement("span");
          fromValue.className = "From";
          fromValue.textContent = transaction.from;
          fromRow.appendChild(fromLabel);
          fromRow.appendChild(fromValue);
      
          // 创建 to 行
          var toRow = document.createElement("div");
          toRow.className = "transaction-row";
          var toLabel = document.createElement("span");
          toLabel.className = "label";
          toLabel.textContent = "To(接收方): ";
          var toValue = document.createElement("span");
          toValue.className = "To";
          toValue.textContent = transaction.to;
          toRow.appendChild(toLabel);
          toRow.appendChild(toValue);
      
          // 创建 value 行
          var valueRow = document.createElement("div");
          valueRow.className = "transaction-row";
          var valueLabel = document.createElement("span");
          valueLabel.className = "label";
          valueLabel.textContent = "Value(交易金额): ";
          var valueValue = document.createElement("span");
          valueValue.className = "value";
          valueValue.textContent = transaction.value;
          valueRow.appendChild(valueLabel);
          valueRow.appendChild(valueValue);
      
          // 将行添加到交易框
          transactionBox.appendChild(fromRow);
          transactionBox.appendChild(toRow);
          transactionBox.appendChild(valueRow);
      
          // 将交易框添加到容器元素
          transactionContainer.appendChild(transactionBox);
        });
        }
        
      }
      
      
      
      
      

    }
  });
});
