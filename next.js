import { useEffect, useState } from "react";
import io from "socket.io-client";
import { useRouter } from "next/router";

const socket = io("http://localhost:8080");

export default function Home() {
  const router = useRouter();
  const [tasks, setTasks] = useState([]);
  const [task, setTask] = useState("");
  const [aiSuggestion, setAiSuggestion] = useState("");
  const [token, setToken] = useState(null);

  useEffect(() => {
    const userToken = localStorage.getItem("token");
    if (!userToken) {
      router.push("/login");
      return;
    }
    setToken(userToken);
    fetch("http://localhost:8080/tasks", {
      headers: { Authorization: `Bearer ${userToken}` },
    })
      .then((res) => res.json())
      .then((data) => setTasks(data));

    socket.on("newTask", (newTask) => {
      setTasks((prev) => [...prev, newTask]);
    });
  }, [router]);

  const createTask = async () => {
    const res = await fetch("http://localhost:8080/tasks", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ title: task, description: "New Task" }),
    });
    const newTask = await res.json();
    setTasks([...tasks, newTask]);
    setTask("");
  };

  const getAISuggestion = async () => {
    const res = await fetch("http://localhost:8080/ai-suggest", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ prompt: "Suggest a new task" }),
    });
    const data = await res.json();
    setAiSuggestion(data.suggestion);
  };

  const logout = () => {
    localStorage.removeItem("token");
    router.push("/login");
  };

  return (
    <div className="p-4 max-w-xl mx-auto">
      <h1 className="text-2xl font-bold">Task Manager</h1>
      <button onClick={logout} className="bg-red-500 text-white p-2 mt-2 w-full">
        Logout
      </button>
      <div className="mt-4">
        <input
          type="text"
          value={task}
          onChange={(e) => setTask(e.target.value)}
          className="border p-2 w-full"
          placeholder="Enter task title"
        />
        <button onClick={createTask} className="bg-blue-500 text-white p-2 mt-2 w-full">
          Add Task
        </button>
      </div>
      <button onClick={getAISuggestion} className="bg-green-500 text-white p-2 mt-2 w-full">
        Get AI Suggestion
      </button>
      {aiSuggestion && <p className="mt-2 p-2 bg-gray-200">AI Suggestion: {aiSuggestion}</p>}
      <ul className="mt-4">
        {tasks.map((t, index) => (
          <li key={index} className="border p-2 mt-2">
            {t.title}
          </li>
        ))}
      </ul>
    </div>
  );
}
