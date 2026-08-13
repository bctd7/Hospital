export interface PatientMockWindow {
  id: string;
  weekday: string;
  date: string;
  session: "上午" | "下午";
  time: string;
  cutoff: string;
}

export interface PatientMockRoom {
  id: string;
  name: string;
  windows: PatientMockWindow[];
}

export interface PatientMockExaminationItem {
  id: string;
  name: string;
  description: string;
  preparation: string;
  rooms: PatientMockRoom[];
}

export interface PatientMockDepartment {
  id: string;
  campus: string;
  name: string;
  items: PatientMockExaminationItem[];
}

export const patientAppointmentMockAdapter = {
  async listDepartments(): Promise<PatientMockDepartment[]> {
    return [
      {
        id: "department-radiology-main",
        campus: "本部院区",
        name: "放射科",
        items: [
          {
            id: "exam-chest-ct",
            name: "胸部 CT",
            description: "用于胸部结构影像检查，实际是否需要增强以医嘱为准。",
            preparation: "请提前移除胸前金属物品，并携带既往影像资料。",
            rooms: [
              {
                id: "room-ct-a",
                name: "CT 室 A",
                windows: [
                  { id: "a-mon-am", weekday: "周一", date: "08-17", session: "上午", time: "09:00—12:00", cutoff: "11:30" },
                  { id: "a-wed-pm", weekday: "周三", date: "08-19", session: "下午", time: "14:00—17:00", cutoff: "16:30" },
                ],
              },
              {
                id: "room-ct-b",
                name: "CT 室 B",
                windows: [
                  { id: "b-tue-am", weekday: "周二", date: "08-18", session: "上午", time: "08:30—11:30", cutoff: "11:00" },
                ],
              },
            ],
          },
          {
            id: "exam-panoramic",
            name: "口腔全景片",
            description: "用于观察牙列、颌骨及相关结构。",
            preparation: "检查前取下眼镜、耳环及可摘义齿。",
            rooms: [{
              id: "room-imaging-c",
              name: "影像室 C",
              windows: [{ id: "c-thu-am", weekday: "周四", date: "08-20", session: "上午", time: "09:00—11:00", cutoff: "10:30" }],
            }],
          },
        ],
      },
      {
        id: "department-radiology-north",
        campus: "北区",
        name: "放射科",
        items: [{
          id: "exam-chest-ct-north",
          name: "胸部 CT",
          description: "与本部执行相同检查项目，由患者选择具体院区和房间。",
          preparation: "请提前移除胸前金属物品。",
          rooms: [{
            id: "room-north-ct",
            name: "北区 CT 室",
            windows: [{ id: "north-fri-pm", weekday: "周五", date: "08-21", session: "下午", time: "13:30—16:30", cutoff: "16:00" }],
          }],
        }],
      },
    ];
  },
};
